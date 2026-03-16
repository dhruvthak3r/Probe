CREATE INDEX idx_results_monitor_time
ON results (monitor_id, created_at DESC);