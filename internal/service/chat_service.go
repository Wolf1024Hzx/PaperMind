package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"wolfden.website/papermind/internal/dto"
	"wolfden.website/papermind/internal/model"
	"wolfden.website/papermind/internal/pkg/prompt"
	"wolfden.website/papermind/internal/repository"
)

type ChatService struct {
	vectorRepo       *repository.VectorRepository
	conversationRepo *repository.ConversationRepository
	embeddingClient  EmbeddingClient
	llmClient        LLMClient
	retrievalTopK    map[string]int
}

func NewChatService(
	vectorRepo *repository.VectorRepository,
	conversationRepo *repository.ConversationRepository,
	embeddingClient EmbeddingClient,
	llmClient LLMClient,
	retrievalTopKExtract int,
	retrievalTopKCompare int,
) *ChatService {
	return &ChatService{
		vectorRepo:       vectorRepo,
		conversationRepo: conversationRepo,
		embeddingClient:  embeddingClient,
		llmClient:        llmClient,
		retrievalTopK: map[string]int{
			"extract": retrievalTopKExtract,
			"compare": retrievalTopKCompare,
		},
	}
}

func (s *ChatService) Ask(ctx context.Context, userID uuid.UUID, req dto.ChatRequest) (*dto.ChatResult, error) {
	// 1. 意图识别
	mode := detectMode(req.Question)

	// 2. 问题向量化
	embeddings, err := s.embeddingClient.Embed(ctx, []string{req.Question})
	if err != nil {
		return nil, fmt.Errorf("问题向量化失败: %w", err)
	}
	queryVector := pgvector.NewVector(embeddings[0])

	// 3. 向量检索
	topK := s.retrievalTopK[mode]

	results, err := s.vectorRepo.Search(ctx, &dto.RetrievalRequest{
		QueryVector:  queryVector,
		UserID:       userID, // 只检索该用户上传的论文
		PaperIDs:     req.PaperIDs,
		SectionTypes: req.SectionTypes,
		YearFrom:     req.YearFrom,
		TopK:         topK,
	})
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	// 如果没有检索结果，返回提示信息
	if len(results) == 0 {
		return &dto.ChatResult{
			Mode:   mode,
			Answer: "在现有论文库中未找到与您问题相关的内容。请尝试上传相关论文后再提问。",
		}, nil
	}

	// 4. 构建 Prompt
	refs := make([]prompt.Reference, len(results))
	for i, r := range results {
		refs[i] = prompt.Reference{
			Index:        i + 1, // 来源编号从 1 开始
			PaperTitle:   r.PaperTitle,
			Authors:      r.Authors,
			SectionTitle: r.SectionTitle,
			PageNumber:   r.PageNumber,
			Content:      r.Content,
		}
	}
	// 选择 System Prompt
	systemPrompt := prompt.ExtractSystemPrompt
	if mode == "compare" {
		systemPrompt = prompt.CompareSystemPrompt
	}
	// 构建 User Prompt（包含检索结果 + 问题）
	userPrompt := prompt.BuildUserPrompt(req.Question, refs)

	// 5. 查询历史消息（如果是追问）
	var historyMessages []ChatMessage
	if req.ConversationID != nil {
		// 先验证对话属于当前用户
		conv, err := s.conversationRepo.FindByIDAndUserID(ctx, *req.ConversationID, userID)
		if err != nil {
			// 对话不存在或不属于当前用户，返回错误
			return nil, fmt.Errorf("对话不存在或无权访问")
		}
		_ = conv // 暂时不需要用到，只是验证权限

		// 查询历史消息
		msgs, err := s.conversationRepo.FindMessages(ctx, *req.ConversationID)
		if err != nil {
			log.Printf("查询历史消息失败: %v", err) // 不影响主流程
		} else {
			for _, m := range msgs {
				historyMessages = append(historyMessages, ChatMessage{
					Role:    m.Role,
					Content: m.Content,
				})
			}
		}
	}

	// 6. 调用 LLM
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, historyMessages...)                             // 加入历史对话
	messages = append(messages, ChatMessage{Role: "user", Content: userPrompt}) // 当前问题

	llmResp, err := s.llmClient.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 7. 保存对话记录
	conversationID, err := s.saveConversation(ctx, userID, req, mode, llmResp, results)
	if err != nil {
		// 保存失败不影响返回，只记录日志
		log.Printf("保存对话记录失败: %v", err)
	}

	// 8. 返回结果
	return &dto.ChatResult{
		ConversationID: conversationID,
		Mode:           mode,
		Answer:         llmResp.Content,
		References:     results,
		TokenUsage: map[string]int{
			"promptTokens":     llmResp.PromptTokens,
			"completionTokens": llmResp.CompletionTokens,
		},
	}, nil
}

// 识别问答模式
func detectMode(question string) string {
	lower := strings.ToLower(question)
	compareKeywords := []string{
		"对比", "区别", "差异", "不同", "比较", "哪个更好",
		"compare", "difference", "versus", "vs", "differ",
	}
	for _, kw := range compareKeywords {
		if strings.Contains(lower, kw) {
			return "compare"
		}
	}
	return "extract"
}

// 保存对话记录到数据库
func (s *ChatService) saveConversation(
	ctx context.Context,
	userID uuid.UUID,
	req dto.ChatRequest,
	mode string,
	llmResp *ChatResponse,
	results []dto.RetrievalResult,
) (uuid.UUID, error) {
	var conversationID uuid.UUID

	// 判断是新建对话还是追加
	if req.ConversationID != nil {
		// 追加到已有对话
		conversationID = *req.ConversationID
		// 更新对话的 updated_at
		if err := s.conversationRepo.UpdateUpdatedAt(ctx, conversationID); err != nil {
			log.Printf("更新对话时间失败: %v", err) // 不影响主流程
		}
	} else {
		// 创建新对话，标题用问题前50字符
		title := req.Question
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		conv := &model.Conversation{
			UserID: userID,
			Title:  &title,
			Mode:   mode,
		}
		if err := s.conversationRepo.Create(ctx, conv); err != nil {
			return uuid.Nil, err
		}
		conversationID = conv.ID
	}

	// 保存用户消息
	userMsg := &model.Message{
		ConversationID: conversationID,
		Role:           "user",
		Content:        req.Question,
	}
	if err := s.conversationRepo.CreateMessage(ctx, userMsg); err != nil {
		return uuid.Nil, err
	}

	// 保存 AI 回答（含引用来源和 token 统计）
	refsJSON, _ := json.Marshal(results)
	refsRaw := json.RawMessage(refsJSON)

	usageJSON, _ := json.Marshal(map[string]int{
		"prompt_tokens":     llmResp.PromptTokens,
		"completion_tokens": llmResp.CompletionTokens,
	})
	usageRaw := json.RawMessage(usageJSON)

	assistantMsg := &model.Message{
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        llmResp.Content,
		ReferencesData: &refsRaw,
		TokenUsage:     &usageRaw,
	}
	if err := s.conversationRepo.CreateMessage(ctx, assistantMsg); err != nil {
		return uuid.Nil, err
	}

	return conversationID, nil
}
