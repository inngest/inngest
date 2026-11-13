SELECT id, data.user.id AS user_id, data.n::VARCHAR AS n_text FROM events WHERE data.user.id = 'u1' AND data.n > 5 AND data.name LIKE 'a%' AND starts_with(data.user.id || '-', 'u') ORDER BY data.n
