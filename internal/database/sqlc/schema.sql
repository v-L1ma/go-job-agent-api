CREATE TABLE jobs (
  id uuid not null,
  plataformJobId text not null,
  title text not null,
  description text not null,
  url text not null,
  isApplied boolean not null,
  status text not null,
  active boolean not null,
  createdBy text not null,
  createdAt timestamp with time zone not null,
  lastModifiedBy text not null,
  lastModifiedAt timestamp with time zone not null,
  platform text not null default ''::text,
  company text not null default ''::text,
  constraint PK_Jobs primary key (Id)
);

CREATE TABLE searchQueries (
  id uuid not null,
  query text not null,
  keywords text[] not null,
  area text not null,
  normalizedhash text not null,
  active boolean not null,
  createdby text not null,
  createdat timestamp with time zone not null,
  lastmodifiedby text not null,
  lastmodifiedat timestamp with time zone not null,
  lastexecutedat timestamp with time zone not null default '-infinity'::timestamp with time zone,
  levels text[] not null default array[]::text[],
  constraint PK_SearchQueries primary key (Id)
);

create index IF not exists IX_SearchQueries_NormalizedHash on searchQueries using btree (NormalizedHash);

CREATE TABLE userCvs (
  id uuid not null,
  userId uuid not null,
  urlFile text not null,
  extractedText text not null,
  active boolean not null,
  createdBy text not null,
  createdAt timestamp with time zone not null,
  lastModifiedBy text not null,
  lastModifiedAt timestamp with time zone not null,
  constraint PK_UserCvs primary key (Id)
);

CREATE TABLE userPreferences (
  id uuid not null,
  userId uuid not null,
  skills text[] not null,
  area text not null,
  active boolean not null,
  createdBy text not null,
  createdAt timestamp with time zone not null,
  lastModifiedBy text not null,
  lastModifiedAt timestamp with time zone not null,
  levels text[] not null default array[]::text[],
  constraint PK_UserPreferences primary key (Id)
);

CREATE TABLE userSearchQueries (
  userId uuid not null,
  searchQueryId uuid not null,
  createdAt timestamp with time zone not null,
  limitedUntil timestamp with time zone not null default '-infinity'::timestamp with time zone,
  savedJobsCount integer not null default 0,
  constraint PK_UserSearchQueries primary key ("UserId", "SearchQueryId")
);

CREATE TABLE generatedCvs (
  id uuid not null,
  userId uuid not null,
  urlFile text not null,
  active boolean not null,
  createdBy text not null,
  createdAt timestamp with time zone not null,
  lastModifiedBy text not null,
  lastModifiedAt timestamp with time zone not null,
  constraint PK_GeneratedCvs primary key ("Id")
);

CREATE TABLE users (
  id uuid not null,
  name text not null,
  CPF text not null,
  email character varying(256) null,
  passwordHash text null,
  accessFailedCount integer not null,
  onboardingCompleted boolean not null default false,
  constraint PK_AspNetUsers primary key ("id") 
);

create index IF not exists "EmailIndex" on users using btree ("NormalizedEmail");