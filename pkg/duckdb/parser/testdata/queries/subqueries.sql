SELECT a FROM t WHERE EXISTS (SELECT 1 FROM other WHERE other.id = t.id);

SELECT (SELECT max(a) FROM t) AS m FROM other;
