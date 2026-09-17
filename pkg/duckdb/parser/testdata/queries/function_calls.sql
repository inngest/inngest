SELECT count(*) FROM t;

SELECT count(DISTINCT a) FROM t;

SELECT sum(a) FILTER (WHERE a > 0) FROM t;

SELECT row_number() OVER (PARTITION BY a ORDER BY b) FROM t;

SELECT row_number() OVER win FROM t;
