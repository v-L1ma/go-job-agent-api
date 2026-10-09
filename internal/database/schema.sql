CREATE EXTENSION IF NOT EXISTS unaccent;

create table public."Jobs" (
  "Id" uuid not null,
  "PlataformJobId" text not null,
  "Title" text not null,
  "Description" text not null,
  "Url" text not null,
  "IsApplied" boolean not null,
  "Status" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  "Platform" text not null default ''::text,
  "Company" text not null default ''::text,
  "TitleEmbedding" public.vector null,
  "DescriptionEmbedding" public.vector null,
  "SearchQueryId" uuid null,
  "ExecutionId" uuid null,
  constraint PK_Jobs primary key ("Id"),
  constraint fk_jobs_searchqueries_searchqueryid foreign KEY ("SearchQueryId") references "SearchQueries" ("Id") on delete set null
) TABLESPACE pg_default;

create table public."SearchQueries" (
  "Id" uuid not null default gen_random_uuid (),
  "Query" text not null,
  "Keywords" text[] not null,
  "Area" text not null,
  "NormalizedHash" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  "LastExecutedAt" timestamp with time zone not null default '-infinity'::timestamp with time zone,
  "Levels" text[] not null default array[]::text[],
  "SearchQueryEmbedding" public.vector null,
  constraint PK_SearchQueries primary key ("Id")
) TABLESPACE pg_default;

create index IF not exists "IX_SearchQueries_NormalizedHash" on public."SearchQueries" using btree ("NormalizedHash") TABLESPACE pg_default;

CREATE TABLE IF NOT EXISTS public."UserCvs" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "UrlFile" text not null,
  "ExtractedText" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_UserCvs primary key (Id)
);

create table public."Preferences" (
  "Id" uuid not null default gen_random_uuid (),
  "UserId" uuid not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  "SimilarityPercent" double precision null default '0.7'::double precision,
  constraint PK_Preferences primary key ("Id")
) TABLESPACE pg_default;

CREATE TABLE IF NOT EXISTS public."UserSearchQueries" (
  "UserId" uuid not null,
  "SearchQueryId" uuid not null,
  "CreatedAt" timestamp with time zone not null,
  "LimitedUntil" timestamp with time zone not null default '-infinity'::timestamp with time zone,
  "SavedJobsCount" integer not null default 0,
  constraint PK_UserSearchQueries primary key ("UserId", "SearchQueryId")
);

CREATE TABLE IF NOT EXISTS public."GeneratedCvs" (
  "Id" uuid not null,
  "UserId" uuid not null,
  "UrlFile" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_GeneratedCvs primary key ("Id")
);

CREATE TABLE IF NOT EXISTS public."AspNetUsers" (
  "Id" uuid not null default gen_random_uuid(),
  "Name" text not null,
  "CPF" text not null,
  "Email" character varying(256) null,
  "PasswordHash" text null,
  "AccessFailedCount" integer not null,
  "OnboardingCompleted" boolean not null default false,
  "LockoutEnd" timestamp with time zone null,
  "LockoutEnabled" boolean not null,
  "TwoFactorEnabled" boolean not null,
  "EmailConfirmed" boolean not null,
  "PhoneNumberConfirmed" boolean not null,
  constraint PK_AspNetUsers primary key ("Id") 
);

create index IF not exists "EmailIndex" on public."AspNetUsers" using btree ("NormalizedEmail");

create table public."JobEvaluations" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "JobId" uuid not null,
  "Liked" boolean not null,
  "Feedback" text null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_JobEvaluations primary key ("Id"),
  constraint FK_JobEvaluations_Jobs_JobId foreign KEY ("JobId") references "Jobs" ("Id") on delete CASCADE
);

create index IF not exists "IX_JobEvaluations_JobId" on public."JobEvaluations" using btree ("JobId") TABLESPACE pg_default;

create table public."CvEvaluations" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "GeneratedCvId" uuid not null,
  "Liked" boolean not null,
  "Feedback" text null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_CvEvaluations primary key ("Id"),
  constraint FK_CvEvaluations_GeneratedCvs_GeneratedCvId foreign KEY ("GeneratedCvId") references "GeneratedCvs" ("Id") on delete CASCADE
);

create index IF not exists "IX_CvEvaluations_GeneratedCvId" on public."CvEvaluations" using btree ("GeneratedCvId") TABLESPACE pg_default;

CREATE TABLE IF NOT EXISTS public."GeneratedCvsNew" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "JobId" uuid not null,
  "FileName" text not null,
  "ExtractedText" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_GeneratedCvs primary key ("Id")
);

CREATE TABLE IF NOT EXISTS public."PasswordResetTokens" (
  "Id" uuid not null default gen_random_uuid(),
  "Email" text not null,
  "TokenHash" text not null,
  "ExpiresAt" timestamp with time zone not null,
  "Used" boolean not null default false,
  "CreatedAt" timestamp with time zone not null default now(),
  constraint PK_PasswordResetTokens primary key ("Id")
);

create index IF not exists IX_PasswordResetTokens_TokenHash on public."PasswordResetTokens" using btree ("TokenHash");

CREATE TABLE IF NOT EXISTS public."Applications" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "JobId" uuid not null,
  "Status" text not null,
  "Obs" text null,
  "Platform" text null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_Applications primary key ("Id"),
  constraint FK_Applications_Users_UserId foreign key ("UserId") references "AspNetUsers" ("Id") on delete CASCADE,
  constraint FK_Applications_Jobs_JobId foreign key ("JobId") references "Jobs" ("Id") on delete CASCADE
);

CREATE INDEX IF NOT EXISTS IX_Applications_UserId ON public."Applications" USING btree ("UserId");
CREATE INDEX IF NOT EXISTS IX_Applications_JobId ON public."Applications" USING btree ("JobId");

create table public."Questions" (
  "Id" uuid not null default gen_random_uuid(),
  "Question" text not null,
  "Type" text null,
  "Options" text[] null,
  "Answer" text not null,
  "JobId" uuid null,
  "UserId" uuid null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_Questions primary key ("Id"),
  constraint Questions_JobId_fkey foreign KEY ("JobId") references "Jobs" ("Id"),
  constraint Questions_UserId_fkey foreign KEY ("UserId") references "AspNetUsers" ("Id")
) TABLESPACE pg_default;

create index IF not exists "IX_Questions_JobId" on public."Questions" using btree ("JobId") TABLESPACE pg_default;

create index IF not exists "IX_Questions_UserId" on public."Questions" using btree ("UserId") TABLESPACE pg_default;

CREATE TABLE IF NOT EXISTS public."ApplicationEvaluations" (
  "Id" uuid not null default gen_random_uuid(),
  "UserId" uuid not null,
  "ApplicationId" uuid not null,
  "Liked" boolean not null,
  "Feedback" text null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint PK_ApplicationEvaluations primary key ("Id"),
  constraint FK_ApplicationEvaluations_Applications_ApplicationId foreign KEY ("ApplicationId") references "Applications" ("Id") on delete CASCADE,
  constraint FK_ApplicationEvaluations_AspNetUsers_UserId foreign KEY ("UserId") references "AspNetUsers" ("Id") on delete CASCADE
);

create table public."ScraperLogs" (
  "Id" uuid not null default gen_random_uuid (),
  "SearchQueryId" uuid null,
  "Query" text not null,
  "Platform" text not null,
  "ExecutedAt" timestamp with time zone not null default now(),
  "Status" text not null,
  "Obs" text null,
  "SavedJobsCount" integer not null default 0,
  "ExecutionId" uuid null,
  constraint pk_scraperlogs primary key ("Id"),
  constraint fk_scraperlogs_searchqueries_searchqueryid foreign KEY ("SearchQueryId") references "SearchQueries" ("Id") on delete set null
) TABLESPACE pg_default;

create table public."Roles" (
  "Id" uuid not null default gen_random_uuid (),
  "Name" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  constraint pk_roles primary key ("Id"),
  constraint uq_roles_name unique ("Name")
) TABLESPACE pg_default;

create table public."UserRoles" (
  "Id" uuid not null default gen_random_uuid (),
  "UserId" uuid not null,
  "RoleId" uuid not null,
  constraint pk_userroles primary key ("UserId", "RoleId"),
  constraint fk_userroles_users_userid foreign key ("UserId") references "AspNetUsers" ("Id") on delete cascade,
  constraint fk_userroles_roles_roleid foreign key ("RoleId") references "Roles" ("Id") on delete cascade
) TABLESPACE pg_default;

CREATE TABLE public."Subscriptions" (
  "Id" UUID PRIMARY KEY default gen_random_uuid (),
  "UserId" UUID NOT NULL,
  "PlanId" UUID NOT NULL,
  "StripeCustomer_id" VARCHAR(255) NOT NULL,
  "StripeSubscription_id" VARCHAR(255) NOT NULL UNIQUE,
  "StripeCheckoutSession_id" VARCHAR(255) UNIQUE,
  "StripePrice_id" VARCHAR(255) NULL,
  "Status" VARCHAR(50) NOT NULL,
  "CurrentPeriodStart" TIMESTAMP,
  "CurrentPeriodEnd" TIMESTAMP,
  "CancelAtPeriodEnd" BOOLEAN NOT NULL DEFAULT FALSE,
  "CanceledAt" TIMESTAMP,
  "CreatedAt" TIMESTAMP NOT NULL DEFAULT NOW(),
  "UpdatedAt" TIMESTAMP NOT NULL DEFAULT NOW(),
  CONSTRAINT fk_subscriptions_users FOREIGN KEY ("UserId") REFERENCES "AspNetUsers" ("Id") ON DELETE CASCADE,
  CONSTRAINT fk_subscriptions_plans FOREIGN KEY ("PlanId") REFERENCES "Plans" ("Id") ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS IX_Subscriptions_UserId ON public."Subscriptions" USING btree ("UserId");

-- Idempotência de webhooks: cada event.id do Stripe é processado no máximo uma vez.
CREATE TABLE IF NOT EXISTS public."StripeWebhookEvents" (
  "Id" VARCHAR(255) NOT NULL PRIMARY KEY,
  "Type" VARCHAR(100) NOT NULL,
  "ProcessedAt" TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Relação persistente usuário da aplicação <-> Stripe Customer.
CREATE TABLE IF NOT EXISTS public."StripeCustomers" (
  "UserId" UUID NOT NULL PRIMARY KEY,
  "StripeCustomerId" VARCHAR(255) NOT NULL UNIQUE,
  "CreatedAt" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "UpdatedAt" TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT fk_stripecustomers_users FOREIGN KEY ("UserId") REFERENCES "AspNetUsers" ("Id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS IX_StripeCustomers_StripeCustomerId ON public."StripeCustomers" USING btree ("StripeCustomerId");

-- Registro de cobranças (invoice.paid / invoice.payment_failed).
-- Sem FK estrita para Subscriptions: o evento pode chegar antes da linha da
-- assinatura existir (ordem de entrega do Stripe não é garantida).
CREATE TABLE IF NOT EXISTS public."StripeInvoices" (
  "StripeInvoiceId" VARCHAR(255) NOT NULL PRIMARY KEY,
  "StripeSubscriptionId" VARCHAR(255) NULL,
  "StripeCustomerId" VARCHAR(255) NULL,
  "AmountPaid" BIGINT NULL,
  "Currency" VARCHAR(10) NULL,
  "Status" VARCHAR(50) NOT NULL,
  "PaidAt" TIMESTAMPTZ NULL,
  "CreatedAt" TIMESTAMPTZ NOT NULL DEFAULT now(),
  "UpdatedAt" TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS IX_StripeInvoices_StripeSubscriptionId ON public."StripeInvoices" USING btree ("StripeSubscriptionId");

CREATE TABLE public."Plans" (
  "Id" UUID PRIMARY KEY default gen_random_uuid (),
  "Name" VARCHAR(255) NOT NULL,
  "StripePrice_id" VARCHAR(255) NULL UNIQUE,
  "StripeProduct_id" VARCHAR(255) NOT NULL UNIQUE,
  "Description" TEXT,
  "Price" integer NOT NULL,
  "RoleId" UUID NOT NULL,
  "Active" BOOLEAN NOT NULL DEFAULT TRUE,
  "CreatedAt" TIMESTAMP NOT NULL DEFAULT NOW(),
  "UpdatedAt" TIMESTAMP NOT NULL DEFAULT NOW()
) TABLESPACE pg_default;

CREATE EXTENSION IF NOT EXISTS unaccent;