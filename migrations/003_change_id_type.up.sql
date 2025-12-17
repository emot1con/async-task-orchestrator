BEGIN;

-- 1. Lepas primary key lama
ALTER TABLE tasks
DROP CONSTRAINT tasks_pkey;

-- 2. Tambah kolom baru sebagai auto increment
ALTER TABLE tasks
ADD COLUMN id_new INTEGER GENERATED ALWAYS AS IDENTITY;

-- 3. Jadikan kolom baru sebagai primary key
ALTER TABLE tasks
ADD CONSTRAINT tasks_pkey PRIMARY KEY (id_new);

-- 4. Hapus kolom id lama (uuid)
ALTER TABLE tasks
DROP COLUMN id;

-- 5. Rename kolom baru menjadi id
ALTER TABLE tasks
RENAME COLUMN id_new TO id;

COMMIT;
