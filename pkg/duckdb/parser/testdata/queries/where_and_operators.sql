SELECT a FROM t WHERE a > 1 AND b < 2 OR c = 3;

SELECT a FROM t WHERE a BETWEEN 1 AND 10;

SELECT a FROM t WHERE a IN (1, 2, 3);

SELECT a FROM t WHERE a IN (SELECT id FROM other);

SELECT a FROM t WHERE a LIKE '%x%';

SELECT a FROM t WHERE a IS NOT NULL;

SELECT a FROM t WHERE a IS DISTINCT FROM b;

SELECT (1 + 2) * 3 - 4 / 2 AS calc FROM t;

SELECT a FROM t WHERE a = NOT b;

SELECT a FROM t WHERE a IN tags AND b NOT IN (tags);

SELECT a FROM t WHERE 'x' = data ->> 'name' AND b <= 1 AND c != 2 AND d=-1;

SELECT a FROM t WHERE 'x' ~~ data ->> 'name' AND b !~* 'y' AND c NOT ~~~ 'z*' AND d ~ 'r' || s;

SELECT a FROM t WHERE (a LIKE 'x') IN (TRUE) AND (b BETWEEN 1 AND 2) NOT BETWEEN FALSE AND TRUE AND c LIKE 'y' ESCAPE (d OR e);

SELECT - -1, -(-a), ~ ~b, a - -1, (c!)! FROM t;
