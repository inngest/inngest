SELECT * FROM sales PIVOT (sum(amount) FOR quarter IN (q1, q2, q3, q4));

SELECT * FROM sales PIVOT (sum(amount) FOR quarter IN (q1, q2) GROUP BY region);

SELECT * FROM wide UNPIVOT (amount FOR quarter IN (q1, q2, q3, q4));

PIVOT sales ON quarter USING sum(amount);

UNPIVOT wide ON q1, q2, q3, q4;
