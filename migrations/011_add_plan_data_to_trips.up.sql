-- Menambahkan kolom plan_data bertipe JSONB
-- JSONB di Postgres sangat powerful (bisa di-query & index), jauh lebih baik dari TEXT biasa.

ALTER TABLE trips
    ADD COLUMN plan_data JSONB NULL;

-- Opsional: Tambahkan index jika nanti kita ingin mencari trip berdasarkan isi plan-nya
-- CREATE INDEX idx_trips_plan_data ON trips USING gin (plan_data);