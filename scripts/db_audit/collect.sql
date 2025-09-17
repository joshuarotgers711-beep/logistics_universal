-- Top queries by total time and mean time
SELECT queryid, calls, total_time, mean_time, rows, query
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 50;

SELECT queryid, calls, total_time, mean_time, rows, query
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 50;

