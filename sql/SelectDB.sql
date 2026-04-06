-- 查看前 10 个向量值
SELECT chunk_index,
       embedding::real[] AS embedding_array
FROM chunks
WHERE paper_id = 'fd124749-0f9c-4a5a-9203-ee71606aded9'
ORDER BY chunk_index
LIMIT 1;