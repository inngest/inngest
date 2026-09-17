SELECT * FROM a JOIN b ON a.id = b.id;

SELECT * FROM a LEFT JOIN b ON a.id = b.id;

SELECT * FROM a LEFT OUTER JOIN b USING (id);

SELECT * FROM a RIGHT JOIN b ON a.id = b.id;

SELECT * FROM a FULL OUTER JOIN b ON a.id = b.id;

SELECT * FROM a CROSS JOIN b;

SELECT * FROM a NATURAL JOIN b;

SELECT * FROM a SEMI JOIN b ON a.id = b.id;

SELECT * FROM a ANTI JOIN b ON a.id = b.id;

SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id;

SELECT * FROM (SELECT id FROM t) sub;

SELECT * FROM a, LATERAL (SELECT * FROM b WHERE b.id = a.id) sub;

SELECT * FROM range(10) t;

SELECT * FROM (a JOIN b ON a.id = b.id) x;

SELECT * FROM a, b;
