-- Rollback (Jika perlu)
-- Ini agak tricky karena user_id sudah berisi string Clerk yang panjang.
-- Kita kembalikan ke UUID hanya jika datanya valid, tapi untuk safety kita biarkan text dulu
-- atau kita kosongkan logika down-nya untuk sementara.

ALTER TABLE trips ALTER COLUMN user_id TYPE UUID USING user_id::uuid;