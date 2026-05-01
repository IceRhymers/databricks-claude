-- Unity Catalog table schemas for databricks-claude OTel ingestion.
-- Replace `your_catalog.your_schema` with your actual catalog and schema.
--
-- These tables receive data from the OTLP proxy embedded in databricks-claude.
-- Each signal (metrics, logs, traces) routes to its own table via the
-- X-Databricks-UC-Table-Name header; a signal is only emitted when its
-- table is configured (see --otel-metrics-table / --otel-logs-table / --otel-traces-table).

-- ---------------------------------------------------------------------------
-- Metrics
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS your_catalog.your_schema.claude_otel_metrics (
  name STRING,
  description STRING,
  unit STRING,
  metric_type STRING,
  gauge STRUCT<
    start_time_unix_nano: LONG,
    time_unix_nano: LONG,
    value: DOUBLE,
    exemplars: ARRAY<STRUCT<
      time_unix_nano: LONG,
      value: DOUBLE,
      span_id: STRING,
      trace_id: STRING,
      filtered_attributes: MAP<STRING, STRING>
    >>,
    attributes: MAP<STRING, STRING>,
    flags: INT
  >,
  sum STRUCT<
    start_time_unix_nano: LONG,
    time_unix_nano: LONG,
    value: DOUBLE,
    exemplars: ARRAY<STRUCT<
      time_unix_nano: LONG,
      value: DOUBLE,
      span_id: STRING,
      trace_id: STRING,
      filtered_attributes: MAP<STRING, STRING>
    >>,
    attributes: MAP<STRING, STRING>,
    flags: INT,
    aggregation_temporality: STRING,
    is_monotonic: BOOLEAN
  >,
  histogram STRUCT<
    start_time_unix_nano: LONG,
    time_unix_nano: LONG,
    count: LONG,
    sum: DOUBLE,
    bucket_counts: ARRAY<LONG>,
    explicit_bounds: ARRAY<DOUBLE>,
    exemplars: ARRAY<STRUCT<
      time_unix_nano: LONG,
      value: DOUBLE,
      span_id: STRING,
      trace_id: STRING,
      filtered_attributes: MAP<STRING, STRING>
    >>,
    attributes: MAP<STRING, STRING>,
    flags: INT,
    min: DOUBLE,
    max: DOUBLE,
    aggregation_temporality: STRING
  >,
  exponential_histogram STRUCT<
    attributes: MAP<STRING, STRING>,
    start_time_unix_nano: LONG,
    time_unix_nano: LONG,
    count: LONG,
    sum: DOUBLE,
    scale: INT,
    zero_count: LONG,
    positive_bucket: STRUCT<
      offset: INT,
      bucket_counts: ARRAY<LONG>
    >,
    negative_bucket: STRUCT<
      offset: INT,
      bucket_counts: ARRAY<LONG>
    >,
    flags: INT,
    exemplars: ARRAY<STRUCT<
      time_unix_nano: LONG,
      value: DOUBLE,
      span_id: STRING,
      trace_id: STRING,
      filtered_attributes: MAP<STRING, STRING>
    >>,
    min: DOUBLE,
    max: DOUBLE,
    zero_threshold: DOUBLE,
    aggregation_temporality: STRING
  >,
  summary STRUCT<
    start_time_unix_nano: LONG,
    time_unix_nano: LONG,
    count: LONG,
    sum: DOUBLE,
    quantile_values: ARRAY<STRUCT<
      quantile: DOUBLE,
      value: DOUBLE
    >>,
    attributes: MAP<STRING, STRING>,
    flags: INT
  >,
  metadata MAP<STRING, STRING>,
  resource STRUCT<
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  resource_schema_url STRING,
  instrumentation_scope STRUCT<
    name: STRING,
    version: STRING,
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  metric_schema_url STRING
) USING DELTA
TBLPROPERTIES (
  'otel.schemaVersion' = 'v1'
);

-- ---------------------------------------------------------------------------
-- Logs
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS your_catalog.your_schema.claude_otel_logs (
  event_name STRING,
  trace_id STRING,
  span_id STRING,
  time_unix_nano LONG,
  observed_time_unix_nano LONG,
  severity_number STRING,
  severity_text STRING,
  body STRING,
  attributes MAP<STRING, STRING>,
  dropped_attributes_count INT,
  flags INT,
  resource STRUCT<
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  resource_schema_url STRING,
  instrumentation_scope STRUCT<
    name: STRING,
    version: STRING,
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  log_schema_url STRING
) USING DELTA
TBLPROPERTIES (
  'otel.schemaVersion' = 'v1'
);

-- ---------------------------------------------------------------------------
-- Traces
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS your_catalog.your_schema.claude_otel_traces (
  trace_id STRING,
  span_id STRING,
  parent_span_id STRING,
  trace_state STRING,
  name STRING,
  kind STRING,
  start_time_unix_nano LONG,
  end_time_unix_nano LONG,
  attributes MAP<STRING, STRING>,
  dropped_attributes_count INT,
  events ARRAY<STRUCT<
    time_unix_nano: LONG,
    name: STRING,
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >>,
  dropped_events_count INT,
  links ARRAY<STRUCT<
    trace_id: STRING,
    span_id: STRING,
    trace_state: STRING,
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT,
    flags: INT
  >>,
  dropped_links_count INT,
  status STRUCT<
    message: STRING,
    code: STRING
  >,
  flags INT,
  resource STRUCT<
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  resource_schema_url STRING,
  instrumentation_scope STRUCT<
    name: STRING,
    version: STRING,
    attributes: MAP<STRING, STRING>,
    dropped_attributes_count: INT
  >,
  scope_schema_url STRING,
  span_schema_url STRING
) USING DELTA
TBLPROPERTIES (
  'otel.schemaVersion' = 'v1'
);
