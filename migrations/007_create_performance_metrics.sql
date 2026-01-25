CREATE TABLE IF NOT EXISTS performance_metrics
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_name   VARCHAR(100), -- 'itinerary', 'logistics', 'total'
    duration_ms INT,
    destination VARCHAR(100),
    model_used  VARCHAR(50),
    created_at  TIMESTAMP        DEFAULT CURRENT_TIMESTAMP
);