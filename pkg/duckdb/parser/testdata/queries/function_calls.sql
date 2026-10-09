SELECT count(*) FROM t;

SELECT count(DISTINCT a) FROM t;

SELECT sum(a) FILTER (WHERE a > 0) FROM t;

SELECT row_number() OVER (PARTITION BY a ORDER BY b) FROM t;

SELECT row_number() OVER win FROM t;

SELECT struct_pack("a b" := 1, "select" := 2, c := 3);

SELECT list_transform(l, "a b" -> 1), list_reduce(l, ("a b", c) -> 1) FROM t;

SELECT [1 FOR "x IN [1]] a, (SELECT 1 FROM u) n, [1 FOR y" IN [1]] c FROM t;
