BEGIN;

-- 1. Lepas primary key integer
ALTER TABLE tasks
DROP CONSTRAINT tasks_pkey;

-- 2. Tambah kolom uuid baru
ALTER TABLE tasks
ADD COLUMN id_old UUID DEFAULT gen_random_uuid();

-- 3. Jadikan uuid sebagai primary key
ALTER TABLE tasks
ADD CONSTRAINT tasks_pkey PRIMARY KEY (id_old);

-- 4. Hapus kolom integer
ALTER TABLE tasks
DROP COLUMN id;

-- 5. Rename kembali ke id
ALTER TABLE tasks
RENAME COLUMN id_old TO id;

COMMIT;
