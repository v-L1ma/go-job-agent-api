-- name: GetJobs :many
SELECT DISTINCT j.*
FROM "Jobs" j
WHERE EXISTS (
    SELECT 1
    FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq
        ON sq."Id" = usq."SearchQueryId"
    CROSS JOIN LATERAL unnest(sq."Keywords") AS keyword
    WHERE usq."UserId" = $1
      AND j."Title" ILIKE '%' || keyword || '%'
);

-- name: GetJobById :one
SELECT "Id", 
    "Title", 
    "Platform", 
    "Company", 
    "Description",
    "Url", 
    "IsApplied", 
    "Status", 
    "Active" 
FROM "Jobs" 
WHERE "Id" = $1;

-- name: ExistsJobEvaluation :one
SELECT EXISTS (
    SELECT 1
    FROM "JobEvaluations"
    WHERE "UserId" = $1 AND "JobId" = $2
) AS "exists";

-- name: EvaluateJob :exec
INSERT INTO "JobEvaluations" ("UserId", "JobId", "Liked", "Feedback", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ExistsCvEvaluation :one
SELECT EXISTS (
    SELECT 1
    FROM "CvEvaluations"
    WHERE "UserId" = $1 AND "GeneratedCvId" = $2
) AS "exists";

-- name: EvaluateCv :exec
INSERT INTO "CvEvaluations" ("UserId", "GeneratedCvId", "Liked", "Feedback", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetUserById :one
SELECT "Id", "Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "LockoutEnd", "LockoutEnabled", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed" 
FROM public."AspNetUsers"
WHERE "Id" = $1;

-- name: GetUserByEmail :one
SELECT "Id", "Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "LockoutEnd", "LockoutEnabled", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed" 
FROM public."AspNetUsers"
WHERE "Email" = $1;

-- name: ExistsUserById :one
SELECT EXISTS (
    SELECT 1
    FROM "AspNetUsers"
    WHERE "Id" = $1
) AS "exists";

-- name: ExistsUserByEmail :one
SELECT EXISTS (
    SELECT 1
    FROM "AspNetUsers"
    WHERE "Email" = $1
) AS "exists";

-- name: CreateUser :exec
INSERT INTO "AspNetUsers" ("Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed", "LockoutEnabled")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: CreateUserPreferences :exec
INSERT INTO "UserPreferences" ("UserId", "Skills", "Levels", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateUserPreferences :exec
UPDATE "UserPreferences" 
SET "UserId" = $1, "Skills" = $2, "Levels" = $3, "Active" = $4, "LastModifiedBy" = $5, "LastModifiedAt" = $6 
WHERE "UserId" = $1;

-- name: FindUserPreferences :one
SELECT EXISTS (
    SELECT 1
    FROM "UserPreferences"
    WHERE "UserId" = $1
) AS "exists";

-- name: GetUserPreferences :many
SELECT "UserId", 
        "Skills", 
        "Levels" 
FROM "UserPreferences" 
WHERE "UserId" = $1 AND "Active" = true;

-- name: SaveUserCv :exec
INSERT INTO "UserCvs" ("UserId", "UrlFile", "ExtractedText", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetUserCv :one
SELECT "UserId", 
"UrlFile", 
"ExtractedText", 
"Active", 
"CreatedBy", 
"CreatedAt", 
"LastModifiedBy", 
"LastModifiedAt" 
FROM "UserCvs" WHERE "UserId" = $1;