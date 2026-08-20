-- name: GetJobs :many

WITH search_config AS (
    SELECT
        sq."Id",
        sq."Keywords",
        sq."Area",
        sq."Levels",
        sq."SearchQueryEmbedding"
    FROM "UserSearchQueries" usq
    JOIN "SearchQueries" sq
        ON sq."Id" = usq."SearchQueryId"
    WHERE usq."UserId" = $1
      AND sq."Active" = true
    LIMIT 1
),
jobs_base AS (
    SELECT
        j.*,
        sq."Keywords",
        sq."Levels",
        sq."SearchQueryEmbedding"

    FROM "Jobs" j
    CROSS JOIN search_config sq
    WHERE NOT EXISTS (
        SELECT 1
        FROM "Applications" a
        WHERE a."JobId" = j."Id"
          AND a."UserId" = $1
    )
    AND NOT EXISTS (
        SELECT 1
        FROM "Applications" a
        WHERE a."JobId" = j."Id"
          AND a."Status" = 'closed'
    )

    AND (
        sqlc.narg('platform')::text IS NULL
        OR j."Platform" = sqlc.narg('platform')::text
    )
),
jobs_scored AS (
    SELECT
        jb.*,
        /*
         * =====================================================
         * EMBEDDING SCORE
         *
         * 70% título
         * 30% descrição
         * =====================================================
         */
        (
            COALESCE(
                (
                    1 - (
                        jb."TitleEmbedding"
                        <=>
                        jb."SearchQueryEmbedding"
                    )
                ) * 0.70,
                0
            )
            +
            COALESCE(
                (
                    1 - (
                        jb."DescriptionEmbedding"
                        <=>
                        jb."SearchQueryEmbedding"
                    )
                ) * 0.30,
                0
            )
        )::double precision AS embedding_similarity,
        /*
         * =====================================================
         * KEYWORD SCORE
         *
         * Keywords no título têm muito mais peso
         * que keywords apenas na descrição.
         * =====================================================
         */
        COALESCE(
            (
                SELECT MAX(
                    CASE

                        /*
                         * Match da frase no título
                         */
                        WHEN
                            to_tsvector(
                                'simple',
                                unaccent(
                                    COALESCE(jb."Title", '')
                                )
                            )
                            @@
                            phraseto_tsquery(
                                'simple',
                                unaccent(keyword)
                            )
                        THEN 1.0

                        /*
                         * Match da frase na descrição
                         */
                        WHEN
                            to_tsvector(
                                'simple',
                                unaccent(
                                    COALESCE(jb."Description", '')
                                )
                            )
                            @@
                            phraseto_tsquery(
                                'simple',
                                unaccent(keyword)
                            )
                        THEN 0.4

                        ELSE 0.0

                    END
                )

                FROM unnest(jb."Keywords") AS keyword
            ),
            0
        )::double precision AS keyword_score,
        /*
         * =====================================================
         * LEVEL SCORE
         *
         * Também é um BOOST, não um filtro.
         * =====================================================
         */
        COALESCE(
            (
                SELECT MAX(
                    CASE
                        /*
                         * Nível no título
                         */
                        WHEN
                            to_tsvector(
                                'simple',
                                unaccent(
                                    COALESCE(jb."Title", '')
                                )
                            )
                            @@
                            phraseto_tsquery(
                                'simple',
                                unaccent(level)
                            )
                        THEN 1.0
                        /*
                         * Nível na descrição
                         */
                        WHEN
                            to_tsvector(
                                'simple',
                                unaccent(
                                    COALESCE(jb."Description", '')
                                )
                            )
                            @@
                            phraseto_tsquery(
                                'simple',
                                unaccent(level)
                            )
                        THEN 0.3
                        ELSE 0.0
                    END
                )
                FROM unnest(jb."Levels") AS level
            ),
            0
        )::double precision AS level_score
    FROM jobs_base jb
),
final_jobs AS (
    SELECT
        *,
        /*
         * =====================================================
         * FINAL SCORE
         *
         * 70% = semântica
         * 20% = keywords
         * 10% = nível
         * =====================================================
         */
        (
            embedding_similarity * 0.70
            +
            keyword_score * 0.20
            +
            level_score * 0.10
        )::double precision AS similarity
    FROM jobs_scored
)
SELECT *
FROM final_jobs
WHERE (
    sqlc.narg('cursor_similarity')::float IS NULL
    OR (
        similarity,
        "CreatedAt",
        "Id"
    )
    <
    (
        sqlc.narg('cursor_similarity')::float,
        sqlc.narg('cursor_created_at')::timestamptz,
        sqlc.narg('cursor_id')::uuid
    )
)
AND (
    sqlc.narg('min_similarity')::float IS NULL
    OR similarity >= sqlc.narg('min_similarity')::float
)
AND NOT EXISTS (
    SELECT 1
    FROM "JobEvaluations" je
    WHERE je."UserId" = $1
      AND je."JobId" = final_jobs."Id"
      AND je."Liked" = false
)
ORDER BY
    similarity DESC,
    "CreatedAt" DESC,
    "Id" DESC
LIMIT $2;

-- name: GetJobsWithoutEmbeddings :many
SELECT j."Id",
  j."Title",
  j."Description"
FROM "Jobs" j
WHERE j."TitleEmbedding" IS NULL
ORDER BY "CreatedAt" DESC
LIMIT 1000;

-- name: AddEmbeddings :exec
UPDATE "Jobs"
SET "TitleEmbedding" = $1, "DescriptionEmbedding" = $2
WHERE "Id" = $3;

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
SELECT u."Id", u."Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "LockoutEnd", "LockoutEnabled", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed", COALESCE(r."Name", 'Standard') AS "RoleName"
FROM public."AspNetUsers" as u
LEFT JOIN "UserRoles" ur ON ur."UserId" = u."Id"
LEFT JOIN "Roles" r ON r."Id" = ur."RoleId"
WHERE u."Id" = $1;

-- name: GetUserByEmail :one
SELECT u."Id", u."Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "LockoutEnd", "LockoutEnabled", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed", COALESCE(r."Name", 'Standard') AS "RoleName"
FROM public."AspNetUsers" as u
LEFT JOIN "UserRoles" ur ON ur."UserId" = u."Id"
LEFT JOIN "Roles" r ON r."Id" = ur."RoleId"
WHERE u."Email" = $1;

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

-- name: CreateUser :one
INSERT INTO "AspNetUsers" ("Name", "CPF", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted", "TwoFactorEnabled", "EmailConfirmed", "PhoneNumberConfirmed", "LockoutEnabled")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING "Id";

-- name: CreateUserPreferences :exec
INSERT INTO "Preferences" ("UserId", "SimilarityPercent", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt") 
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdateUserPreferences :exec
UPDATE "Preferences" 
SET "SimilarityPercent" = $2, "Active" = $3, "LastModifiedBy" = $4, "LastModifiedAt" = $5 
WHERE "UserId" = $1;

-- name: FindUserPreferences :one
SELECT EXISTS (
    SELECT 1
    FROM "Preferences"
    WHERE "UserId" = $1
) AS "exists";

-- name: GetUserPreferences :one
SELECT
    u."Id" AS "UserId",
    COALESCE(pref."SimilarityPercent", 0.7) AS "SimilarityPercent",
    COALESCE(sq."Keywords", ARRAY[]::text[]) AS "Keywords",
    COALESCE(sq."Levels", ARRAY[]::text[]) AS "Levels"
FROM "AspNetUsers" u
LEFT JOIN "Preferences" pref
    ON pref."UserId" = u."Id"
    AND pref."Active" = true
LEFT JOIN "UserSearchQueries" usq
    ON usq."UserId" = u."Id"
LEFT JOIN "SearchQueries" sq
    ON sq."Id" = usq."SearchQueryId"
WHERE u."Id" = $1;

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
INSERT INTO "Questions" ("UserId", "JobId", "Question", "Type", "Options", "Answer", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

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

-- name: GetApplicationsWithQuestions :many
SELECT
    a."Id",
    j."Title",
    j."Company",
    a."Platform",
    a."Status",
    a."CreatedAt",
    COUNT(q."Id")::int AS "QuestionsCount",
    COALESCE(
        json_agg(
            CASE WHEN q."Id" IS NOT NULL THEN
                json_build_object(
                    'id', q."Id",
                    'question', q."Question",
                    'answer', q."Answer",
                    'type', q."Type",
                    'options', q."Options"
                )
            END
        ) FILTER (WHERE q."Id" IS NOT NULL),
        '[]'::json
    ) AS "Questions"
FROM "Applications" AS a
LEFT JOIN "Jobs" AS j ON j."Id" = a."JobId"
LEFT JOIN "Questions" AS q ON q."JobId" = a."JobId" AND q."UserId" = a."UserId" AND q."Active" = true
WHERE a."UserId" = $1 AND a."Status" != 'closed'
GROUP BY a."Id", j."Title", j."Company", a."Platform", a."Status", a."CreatedAt"
ORDER BY a."CreatedAt" DESC;

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
INSERT INTO "SearchQueries" ("Query", "Keywords", "NormalizedHash", "Area", "Levels", "Active", "SearchQueryEmbedding", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING "Id";

-- name: UpdateSearchQuery :exec
UPDATE "SearchQueries" SET "Query" = $1, "Keywords" = $2, "NormalizedHash" = $3, "Levels" = $4, "SearchQueryEmbedding" = $5, "LastModifiedBy" = $6, "LastModifiedAt" = $7
FROM "UserSearchQueries" AS usq
WHERE usq."UserId" = $8 AND usq."SearchQueryId" = "SearchQueries"."Id";

-- name: ExistsSearchQueryByUserId :one
SELECT EXISTS (
    SELECT 1
    FROM "UserSearchQueries"
    WHERE "UserId" = $1
) AS "exists";

-- name: GetSearchQueryIdByUserId :one
SELECT "SearchQueryId" FROM "UserSearchQueries"
WHERE "UserId" = $1;

-- name: ExistsSearchQueryByAnotherUser :one
SELECT EXISTS (
    SELECT 1
    FROM "UserSearchQueries"
    WHERE "SearchQueryId" = $1
    AND "UserId" != $2
) AS "exists";

-- name: GetSearchQueryByUserId :one
SELECT sq."Id", sq."Query", sq."Keywords", sq."NormalizedHash", sq."Levels"
FROM "SearchQueries" AS sq
LEFT JOIN "UserSearchQueries" AS usq ON usq."SearchQueryId" = sq."Id"
WHERE usq."UserId" = $1;

-- name: GetScrapperLogs :many
SELECT "Id", "Query", "Platform", "ExecutedAt", "Status", "Obs", "SavedJobsCount"
FROM "ScraperLogs"
ORDER BY "ExecutedAt" DESC;

-- name: GetSearchQueries :many
SELECT
    sq."Id",
    sq."Query",
    sq."Keywords",
    sq."NormalizedHash",
    sq."Levels",
    COALESCE(sl."Executions", 0) AS "Executions",
    sl."LastExecutedAt"
FROM "SearchQueries" sq
LEFT JOIN (
    SELECT
        "SearchQueryId",
        COUNT(*) AS "Executions",
        MAX("ExecutedAt") AS "LastExecutedAt"
    FROM "ScraperLogs"
    GROUP BY "SearchQueryId"
) sl
    ON sl."SearchQueryId" = sq."Id"
ORDER BY sl."LastExecutedAt" DESC NULLS LAST;

-- name: GetSearchQueryScrapperLogs :many
SELECT "Id", "Query", "Platform", "ExecutedAt", "Status", "Obs", "SavedJobsCount", "ExecutionId"
FROM "ScraperLogs"
WHERE "SearchQueryId" = $1;

-- name: GetExecutionJobs :many
SELECT j."Id", j."Title", j."Url", j."Platform", j."Company"
FROM "Jobs" j
INNER JOIN "ScraperLogs" sl ON sl."ExecutionId" = j."ExecutionId"
WHERE j."ExecutionId" = $1;

-- name: CreateRole :exec
INSERT INTO "Roles" ("Name", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6);

-- name: UpdateRole :exec
UPDATE "Roles"
SET "Name" = $1, "Active" = $2, "LastModifiedBy" = $3, "LastModifiedAt" = $4
WHERE "Id" = $5;

-- name: GetRoles :many
SELECT "Id", "Name", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt"
FROM "Roles";

-- name: ExistsRoleByName :one
SELECT EXISTS (
    SELECT 1
    FROM "Roles"
    WHERE "Name" = $1 AND (
        sqlc.narg('id')::uuid IS NULL
        OR "Id" != sqlc.narg('id')::uuid
    )
) AS "exists";

-- name: ExistsRoleById :one
SELECT EXISTS (
    SELECT 1
    FROM "Roles"
    WHERE "Id" = $1
) AS "exists";

-- name: GetRoleByName :one
SELECT "Id", "Name", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt"
FROM "Roles"
WHERE "Name" = $1;

-- name: GetRoleById :one
SELECT "Id", "Name", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt"
FROM "Roles"
WHERE "Id" = $1;

-- name: DeactivateRole :exec
UPDATE "Roles"
SET "Active" = false, "LastModifiedBy" = $2, "LastModifiedAt" = $3
WHERE "Id" = $1;

-- name: AssignRoleToUser :exec
INSERT INTO "UserRoles" ("UserId", "RoleId")
VALUES ($1, $2);