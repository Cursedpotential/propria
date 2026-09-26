-- images table
create table if not exists images (
  id uuid primary key default gen_random_uuid(),
  filename text,
  supabase_url text,
  r2_url text,
  exif_metadata jsonb,
  ocr_text text,
  notes text,
  created_at timestamp default now(),
  timeline_date timestamp,
  account_id uuid references accounts(id),
  source text
);

-- accounts table
create table if not exists accounts (
  id uuid primary key default gen_random_uuid(),
  user_email text,
  account_type text,
  created_at timestamp default now()
);

-- pipeline logs (for rekognition / OCR)
create table if not exists uploads (
  id uuid primary key default gen_random_uuid(),
  image_id uuid references images(id),
  status text,
  pipeline_logs jsonb,
  created_at timestamp default now()
);

-- optional tagging system
create table if not exists tags (
  id uuid primary key default gen_random_uuid(),
  image_id uuid references images(id),
  tag text
);
