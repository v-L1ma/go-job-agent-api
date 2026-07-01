CREATE TABLE IF NOT EXISTS public."Jobs" (
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
  constraint PK_Jobs primary key (Id)
);

CREATE TABLE IF NOT EXISTS public."SearchQueries" (
  "Id" uuid not null,
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
  constraint PK_SearchQueries primary key (Id)
);

create index IF not exists IX_SearchQueries_NormalizedHash on public."SearchQueries" using btree (NormalizedHash);

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

CREATE TABLE IF NOT EXISTS public."UserPreferences" (
  "Id" uuid not null,
  "UserId" uuid not null,
  "Skills" text[] not null,
  "Area" text not null,
  "Active" boolean not null,
  "CreatedBy" text not null,
  "CreatedAt" timestamp with time zone not null,
  "LastModifiedBy" text not null,
  "LastModifiedAt" timestamp with time zone not null,
  "Levels" text[] not null default array[]::text[],
  constraint PK_UserPreferences primary key (Id)
);

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