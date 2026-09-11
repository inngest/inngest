SELECT row_number() OVER win FROM t WINDOW win AS (PARTITION BY a ORDER BY b);

SELECT a, row_number() OVER (ORDER BY a) AS rn FROM t QUALIFY rn = 1;
