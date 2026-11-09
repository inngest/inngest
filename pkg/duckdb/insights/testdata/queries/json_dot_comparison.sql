SELECT id, data.user.id AS user_id FROM events WHERE data.user.id = 'u1' AND data.n > 5 AND data.name LIKE 'a%' ORDER BY data.n
