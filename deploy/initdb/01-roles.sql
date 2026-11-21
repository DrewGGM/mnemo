-- Runs once, as the PostgreSQL superuser, when the data directory is empty.
-- Passwords come from the environment so that Compose and CI each set their own.
\set ON_ERROR_STOP on
\getenv admin_password MNEMO_ADMIN_DB_PASSWORD
\getenv app_password MNEMO_APP_DB_PASSWORD

-- mnemo_admin owns the schema and runs migrations and admin subcommands.
-- FORCE ROW LEVEL SECURITY also binds a table's owner, so it needs BYPASSRLS;
-- CREATEDB lets tests create throwaway databases (ADR-0007, decision 2).
CREATE ROLE mnemo_admin LOGIN BYPASSRLS CREATEDB PASSWORD :'admin_password';

-- mnemo_app is the service role: no BYPASSRLS, owns nothing, and gets table
-- privileges one by one from the migrations. Since PostgreSQL 15, PUBLIC has
-- no CREATE on schema public, and public belongs to the database owner.
CREATE ROLE mnemo_app LOGIN NOBYPASSRLS PASSWORD :'app_password';

ALTER DATABASE :"DBNAME" OWNER TO mnemo_admin;

-- pgvector is not a trusted extension, so only a superuser can create it.
-- Creating it in template1 too gives it to every database mnemo_admin creates
-- later, such as the throwaway ones of the integration tests.
CREATE EXTENSION vector;
\connect template1
CREATE EXTENSION vector;
