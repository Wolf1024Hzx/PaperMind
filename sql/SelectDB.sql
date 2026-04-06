-- 查看 chunks 数量
SELECT COUNT(*) FROM chunks WHERE paper_id = '08423efa-9b7f-43b7-b838-bc6458a4646a';

-- 查看每个 chunk 的基本信息
SELECT chunk_index, section_type, section_title, token_count,
       LEFT(content, 50) AS content_preview
FROM chunks
WHERE paper_id = '08423efa-9b7f-43b7-b838-bc6458a4646a'
ORDER BY chunk_index;

-- 验证 embedding 列非空
SELECT chunk_index, array_length(embedding::real[], 1) AS dim
FROM chunks
WHERE paper_id = '08423efa-9b7f-43b7-b838-bc6458a4646a'
LIMIT 5;