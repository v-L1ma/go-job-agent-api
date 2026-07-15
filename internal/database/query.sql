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
) AND NOT EXISTS (
    SELECT 1
    FROM "Applications" a
    WHERE a."JobId" = j."Id"
) AND (
    sqlc.narg('cursor')::timestamptz IS NULL
    OR
    j."CreatedAt" < sqlc.narg('cursor')::timestamptz
)
ORDER BY "CreatedAt" DESC
LIMIT $2;

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

-- name: GetGeneratedCvById :one
SELECT "UserId", "JobId", "FileName", "ExtractedText"
FROM "GeneratedCvsNew" WHERE "Id" = $1;

-- name: GetGeneratedCvs :many
SELECT "UserId", "JobId", j."Title", cv."FileName", "ExtractedText"
FROM "GeneratedCvsNew" as cv
LEFT JOIN "Jobs" as j ON cv."JobId" = j."Id" 
WHERE "UserId" = $1;

-- name: SaveGeneratedCV :exec
INSERT INTO "GeneratedCvsNew" ("UserId", "JobId", "FileName", "ExtractedText", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: UpdateUser :exec
UPDATE "AspNetUsers"
SET "Name" = $2, "CPF" = $3, "Email" = $4
WHERE "Id" = $1;

-- name: UpdateUserPassword :exec
UPDATE "AspNetUsers"
SET "PasswordHash" = $2
WHERE "Id" = $1;

-- name: GetUserJobStats :one
SELECT
    COUNT(*)::int AS total,
    COUNT(*) FILTER (WHERE j."IsApplied" = true)::int AS applied,
    COUNT(*) FILTER (WHERE j."IsApplied" = false)::int AS skipped
FROM "Jobs" j
WHERE EXISTS (
    SELECT 1 FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq ON sq."Id" = usq."SearchQueryId"
    CROSS JOIN LATERAL unnest(sq."Keywords") AS keyword
    WHERE usq."UserId" = $1
    AND j."Title" ILIKE '%' || keyword || '%'
);

-- name: GetPrevMonthJobCount :one
SELECT COUNT(*)::int AS count
FROM "Jobs" j
WHERE EXISTS (
    SELECT 1 FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq ON sq."Id" = usq."SearchQueryId"
    CROSS JOIN LATERAL unnest(sq."Keywords") AS keyword
    WHERE usq."UserId" = $1
    AND j."Title" ILIKE '%' || keyword || '%'
)
AND j."CreatedAt" >= date_trunc('month', CURRENT_DATE - INTERVAL '1 month')
AND j."CreatedAt" < date_trunc('month', CURRENT_DATE);

-- name: GetApplicationsPerDay :many
SELECT j."CreatedAt"::date AS date, COUNT(*)::int AS count
FROM "Jobs" j
WHERE EXISTS (
    SELECT 1 FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq ON sq."Id" = usq."SearchQueryId"
    CROSS JOIN LATERAL unnest(sq."Keywords") AS keyword
    WHERE usq."UserId" = $1
    AND j."Title" ILIKE '%' || keyword || '%'
)
AND j."CreatedAt" >= CURRENT_DATE - INTERVAL '6 days'
GROUP BY j."CreatedAt"::date
ORDER BY j."CreatedAt"::date;

-- name: GetPlatformDistribution :many
SELECT j."Platform" AS platform, COUNT(*)::int AS count
FROM "Jobs" j
WHERE EXISTS (
    SELECT 1 FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq ON sq."Id" = usq."SearchQueryId"
    CROSS JOIN LATERAL unnest(sq."Keywords") AS keyword
    WHERE usq."UserId" = $1
    AND j."Title" ILIKE '%' || keyword || '%'
)
GROUP BY j."Platform"
ORDER BY COUNT(*) DESC;

-- name: CreatePasswordResetToken :exec
INSERT INTO "PasswordResetTokens" ("Email", "TokenHash", "ExpiresAt")
VALUES ($1, $2, $3);

-- name: GetPasswordResetTokenByHash :one
SELECT "Id", "Email", "TokenHash", "ExpiresAt", "Used", "CreatedAt"
FROM "PasswordResetTokens"
WHERE "TokenHash" = $1 AND "Used" = false AND "ExpiresAt" > now()
LIMIT 1;

-- name: MarkResetTokenAsUsed :exec
UPDATE "PasswordResetTokens" SET "Used" = true WHERE "Id" = $1;

-- name: UpdateUserPasswordByEmail :exec
UPDATE "AspNetUsers" SET "PasswordHash" = $2 WHERE "Email" = $1;

-- name: CreateApplication :exec
INSERT INTO "Applications" ("UserId", "JobId", "Status", "Obs", "Platform", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CreateQuestion :exec
INSERT INTO "Questions" ("UserId", "JobId", "Question", "Answer", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: FindQuestionAnswer :many
SELECT "Question", 
    "Answer"
FROM "Questions"
WHERE "Question" = ANY(sqlc.arg(questions)::text[]);

-- name: GetUserQuestions :many
SELECT q."Question",
    q."Answer",
    j."Title"
FROM "Questions" AS q
LEFT JOIN "Jobs" AS j ON j."Id" = q."JobId"
WHERE "UserId" = $1 AND q."Active" = true;

-- name: GetQuestionById :one
SELECT q."Question",
    q."Answer",
    q."UserId"
FROM "Questions" AS q
WHERE "Id" = $1 AND q."Active" = true;

-- name: UpdateQuestionAnswer :exec
UPDATE "Questions" SET "Answer" = $2 WHERE "Id" = $1;

-- name: GetUserApplications :many
SELECT a."Status",
    a."Obs",
    a."Platform",
    j."Title",
    a."CreatedAt"
FROM "Applications" AS a
LEFT JOIN "Jobs" AS j ON j."Id" = a."JobId"
WHERE a."UserId" = $1;

-- name: CreateApplicationRate :exec
INSERT INTO "ApplicationEvaluations" ("UserId", "ApplicationId", "Liked", "Feedback", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: CreateUserSearchQuery :exec
INSERT INTO "UserSearchQueries" ("UserId", "SearchQueryId", "CreatedAt")
VALUES ($1, $2, $3);

-- name: UpdateUserSearchQuery :exec
UPDATE "UserSearchQueries" SET "SearchQueryId" = $2
WHERE "UserId" = $1;

-- name: CreateSearchQuery :one
INSERT INTO "SearchQueries" ("Query", "Keywords", "NormalizedHash", "Area", "Levels", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING "Id";

-- name: UpdateSearchQuery :exec
UPDATE "SearchQueries" SET "Query" = $1, "Keywords" = $2, "NormalizedHash" = $3, "Levels" = $4, "LastModifiedBy" = $5, "LastModifiedAt" = $6
FROM "UserSearchQueries" AS usq
WHERE usq."UserId" = $7 AND usq."SearchQueryId" = "SearchQueries"."Id";

-- name: ExistsSearchQueryByUserId :one
SELECT EXISTS (
    SELECT 1
    FROM "UserSearchQueries"
    WHERE "UserId" = $1
) AS "exists";

-- name: ExistsSearchQueryByAnotherUser :one
SELECT EXISTS (
    SELECT 1
    FROM "SearchQueries"
    WHERE "NormalizedHash" = $1
    AND "Id" IN (
        SELECT "SearchQueryId"
        FROM "UserSearchQueries"
        WHERE "UserId" != $2
    )
) AS "exists";

-- name: GetSearchQueryByUserId :one
SELECT sq."Id", sq."Query", sq."Keywords", sq."NormalizedHash", sq."Levels"
FROM "SearchQueries" AS sq
LEFT JOIN "UserSearchQueries" AS usq ON usq."SearchQueryId" = sq."Id"
WHERE usq."UserId" = $1;