-- +goose Up
INSERT INTO products.macro_categories (id, name, slug) VALUES
    ('ac000000-0000-0000-0000-000000000001', 'Hamburguesas',    'hamburguesas'),
    ('ac000000-0000-0000-0000-000000000002', 'Pizzas',          'pizzas'),
    ('ac000000-0000-0000-0000-000000000003', 'Bebidas',         'bebidas'),
    ('ac000000-0000-0000-0000-000000000004', 'Combos',          'combos'),
    ('ac000000-0000-0000-0000-000000000005', 'Perros Calientes','perros-calientes'),
    ('ac000000-0000-0000-0000-000000000006', 'Papas',           'papas'),
    ('ac000000-0000-0000-0000-000000000007', 'Fritangas',       'fritangas'),
    ('ac000000-0000-0000-0000-000000000008', 'Parrillas',       'parrillas'),
    ('ac000000-0000-0000-0000-000000000009', 'Arepas',          'arepas'),
    ('ac000000-0000-0000-0000-000000000010', 'Lasañas',         'lasanas'),
    ('ac000000-0000-0000-0000-000000000011', 'Sandwiches',      'sandwiches'),
    ('ac000000-0000-0000-0000-000000000012', 'Variedades',      'variedades'),
    ('ac000000-0000-0000-0000-000000000013', 'Adiciones',       'adiciones');

-- Hamburguesas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000001'
WHERE id IN (
    'df0b89f7-d866-42ca-b80b-cc9b390a434d', -- Hamburguesas (D.M.O Delicias)
    'bbd7ba77-fa1e-4807-bb42-09ee1a789e49', -- Smash Burgers (DuckTruck22)
    '902f2bd2-92b2-44b2-8267-6742341fb181'  -- Hamburguesas (Goofy Delicias)
);

-- Pizzas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000002'
WHERE id IN (
    '7bbb1577-713f-4ab2-b6af-afbe8b7e43b1'  -- Goofy Pizzas (Goofy Delicias)
);

-- Bebidas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000003'
WHERE id IN (
    '6d18176e-3e60-497e-8e11-81f1c2db3d6c', -- Bebidas (D.M.O Delicias)
    '2b02c0b8-00a4-4c84-a43f-561f6efb887d', -- Bebidas (DuckTruck22)
    '0d7806a5-5401-453b-9a9b-abe54b7b8536', -- Bebidas (Goofy Delicias)
    '86a248eb-620e-42af-a10e-ee03cc5c740c', -- Cócteles (D.M.O Delicias)
    'e746cec0-6e7a-450d-823d-b388b5b2292a', -- Cervezas (Goofy Delicias)
    '16260554-e575-41b3-903b-24134386b088', -- Jugos Naturales (Goofy Delicias)
    'd1006b99-9e8f-461b-bd17-166561383ab9'  -- Refrescantes (Goofy Delicias)
);

-- Combos
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000004'
WHERE id IN (
    '27ae6a9c-6eff-4123-9c2f-337365047d5c', -- Combos (D.M.O Delicias)
    '3a446280-861d-4950-b287-e3068e7327b9', -- Combos (DuckTruck22)
    '6b55798e-dbe7-4f2d-b925-fd0976b98e30'  -- Combos (Goofy Delicias)
);

-- Perros Calientes
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000005'
WHERE id IN (
    '00aaf1a4-5ce2-44c9-b74f-42ed84189c12', -- Perros Calientes (D.M.O Delicias)
    '34baec13-f017-404c-8464-21904c63ae7a'  -- Perros Calientes (Goofy Delicias)
);

-- Papas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000006'
WHERE id IN (
    '26a72eec-3e0e-4f13-803f-43fb7dbb2152', -- Papas Fritas (DuckTruck22)
    '5f06b519-f384-4a1e-8cd2-2d69083d58a1', -- Papas Y Pollos (D.M.O Delicias)
    '783ffad7-c861-4548-898c-70b145a89926', -- Salchipapas (D.M.O Delicias)
    '6467550f-fb8a-4383-9d80-8544d80a8432'  -- Salchipapas (Goofy Delicias)
);

-- Fritangas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000007'
WHERE id IN (
    '96da02c0-b7db-48f5-ba0b-7fa2d49bf3f7', -- Fritangas (D.M.O Delicias)
    '0df00fd8-1a7c-4bf6-8bb9-96439c5cbc87'  -- Goofy Antojos (Goofy Delicias)
);

-- Parrillas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000008'
WHERE id IN (
    '7e441237-a818-42d2-bb54-9b8747198305'  -- Parrillas (Goofy Delicias)
);

-- Arepas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000009'
WHERE id IN (
    'a584981a-7a5e-49f9-88fa-2cd950d06dc8'  -- Goofy Arepas (Goofy Delicias)
);

-- Lasañas
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000010'
WHERE id IN (
    '6780ad87-bbc8-45d5-962e-79a3f33b43ba'  -- Lasañas (Goofy Delicias)
);

-- Sandwiches
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000011'
WHERE id IN (
    'b375ade0-04a7-45fb-bbb8-eb7281e3e97e'  -- Sandwich Cubano (Goofy Delicias)
);

-- Variedades
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000012'
WHERE id IN (
    'f659ff8e-232c-4f3a-883c-dc925357a368'  -- Más Delicias (Goofy Delicias)
);

-- Adiciones
UPDATE products.categories SET macro_category_id = 'ac000000-0000-0000-0000-000000000013'
WHERE id IN (
    'bbfacff7-7fb1-4efe-8497-d8f3213a9522'  -- Adiciones (Goofy Delicias)
);

-- +goose Down
UPDATE products.categories SET macro_category_id = NULL;
DELETE FROM products.macro_categories
WHERE id::text LIKE 'ac000000-0000-0000-0000-%';
