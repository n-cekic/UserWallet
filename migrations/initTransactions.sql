-- Create the database
CREATE DATABASE transactions;

-- Connect to the newly created database
\c transactions;

-- Crea the 'userbalance' table
CREATE TABLE "userbalance" (
    user_id SERIAL PRIMARY KEY,
    balance VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Crea the 'transaction' table
CREATE TABLE "transaction" (
    id SERIAL PRIMARY KEY,
    "type" VARCHAR(255) NOT NULL,
    "from" INTEGER NOT NULL,
    "to" INTEGER NOT NULL,
    "amount" DOUBLE PRECISION,
    "timestamp" TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);