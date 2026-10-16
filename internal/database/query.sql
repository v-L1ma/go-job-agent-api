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
    /*
     * Filtros de texto (case-insensitive) aplicados AQUI, na mesma CTE onde
     * o "platform" já é filtrado — ou seja, ANTES da comparação do cursor
     * (que acontece em final_jobs). Assim a paginação por cursor continua
     * estável: o cursor sempre percorre o MESMO conjunto filtrado.
     *
     * "stack" casa com o TÍTULO da vaga (o rótulo da UI é "Cargo ou Stack").
     */
    AND (
        sqlc.narg('stack')::text IS NULL
        OR j."Title" ILIKE '%' || sqlc.narg('stack')::text || '%'
    )
    AND (
        sqlc.narg('company')::text IS NULL
        OR j."Company" ILIKE '%' || sqlc.narg('company')::text || '%'
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
WHERE u."Email" ILIKE $1;

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
SELECT cv."Id", "UserId", "JobId", j."Title", cv."FileName", "ExtractedText", cv."CreatedAt"
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

-- name: CreateEmailConfirmationToken :exec
INSERT INTO "EmailConfirmationTokens" ("Email", "TokenHash", "ExpiresAt")
VALUES ($1, $2, $3);

-- name: GetEmailConfirmationTokenByHash :one
SELECT "Id", "Email", "TokenHash", "ExpiresAt", "Used", "CreatedAt"
FROM "EmailConfirmationTokens"
WHERE "TokenHash" = $1
LIMIT 1;

-- name: MarkEmailConfirmationTokenAsUsed :exec
UPDATE "EmailConfirmationTokens" SET "Used" = true WHERE "Id" = $1;

-- name: ConfirmUserEmailByEmail :exec
UPDATE "AspNetUsers" SET "EmailConfirmed" = true WHERE "Email" = $1;

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
VALUES ($1, $2)
ON CONFLICT ("UserId", "RoleId") DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM "UserRoles"
WHERE "UserId" = $1 AND "RoleId" = $2;

-- name: ExistsUploadedCv :one
SELECT EXISTS(
    SELECT 1 
    FROM "UserCvs" 
    WHERE "UserId" = $1
) AS "exists";

-- name: CreateSubscription :exec
INSERT INTO "Subscriptions" ("Id", "UserId", "PlanId", "StripeCustomer_id", "StripeSubscription_id", "StripeCheckoutSession_id", "StripePrice_id", "Status", "CurrentPeriodStart", "CurrentPeriodEnd", "CancelAtPeriodEnd", "CanceledAt", "CreatedAt", "UpdatedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);

-- name: UpdateSubscription :exec
UPDATE "Subscriptions"
SET "UserId" = $2, "PlanId" = $3, "StripeCustomer_id" = $4, "StripeSubscription_id" = $5, "StripeCheckoutSession_id" = $6, "StripePrice_id" = $7, "Status" = $8, "CurrentPeriodStart" = $9, "CurrentPeriodEnd" = $10, "CancelAtPeriodEnd" = $11, "CanceledAt" = $12, "UpdatedAt" = $13
WHERE "Id" = $1;

-- name: UpsertSubscription :one
INSERT INTO "Subscriptions" (
  "UserId", "PlanId", "StripeCustomer_id", "StripeSubscription_id",
  "StripeCheckoutSession_id", "StripePrice_id", "Status",
  "CurrentPeriodStart", "CurrentPeriodEnd", "CancelAtPeriodEnd",
  "CanceledAt", "CreatedAt", "UpdatedAt"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), $12)
ON CONFLICT ("StripeSubscription_id") DO UPDATE SET
  "UserId" = EXCLUDED."UserId",
  "PlanId" = EXCLUDED."PlanId",
  "StripeCustomer_id" = EXCLUDED."StripeCustomer_id",
  "StripeCheckoutSession_id" = COALESCE(EXCLUDED."StripeCheckoutSession_id", "Subscriptions"."StripeCheckoutSession_id"),
  "StripePrice_id" = COALESCE(EXCLUDED."StripePrice_id", "Subscriptions"."StripePrice_id"),
  "Status" = EXCLUDED."Status",
  "CurrentPeriodStart" = COALESCE(EXCLUDED."CurrentPeriodStart", "Subscriptions"."CurrentPeriodStart"),
  "CurrentPeriodEnd" = COALESCE(EXCLUDED."CurrentPeriodEnd", "Subscriptions"."CurrentPeriodEnd"),
  "CancelAtPeriodEnd" = EXCLUDED."CancelAtPeriodEnd",
  "CanceledAt" = COALESCE(EXCLUDED."CanceledAt", "Subscriptions"."CanceledAt"),
  "UpdatedAt" = EXCLUDED."UpdatedAt"
RETURNING "Id";

-- name: SetSubscriptionCheckoutSession :execrows
UPDATE "Subscriptions"
SET "StripeCheckoutSession_id" = $2,
    "StripeCustomer_id" = CASE WHEN sqlc.narg(customer_id)::text IS NOT NULL THEN sqlc.narg(customer_id)::text ELSE "StripeCustomer_id" END,
    "UpdatedAt" = NOW()
WHERE "StripeSubscription_id" = $1;

-- name: InsertSubscriptionFromCheckout :execrows
INSERT INTO "Subscriptions" (
  "UserId", "PlanId", "StripeCustomer_id", "StripeSubscription_id",
  "StripeCheckoutSession_id", "Status", "CancelAtPeriodEnd", "CreatedAt", "UpdatedAt"
)
VALUES ($1, $2, $3, $4, $5, 'incomplete', FALSE, NOW(), NOW())
ON CONFLICT ("StripeSubscription_id") DO NOTHING;

-- name: GetSubscriptionByUserId :one
SELECT "Id", "UserId", "PlanId", "StripeCustomer_id", "StripeSubscription_id", "StripeCheckoutSession_id", "StripePrice_id", "Status", "CurrentPeriodStart", "CurrentPeriodEnd", "CancelAtPeriodEnd", "CanceledAt", "CreatedAt", "UpdatedAt"
FROM "Subscriptions"
WHERE "UserId" = $1
ORDER BY "CreatedAt" DESC, "UpdatedAt" DESC
LIMIT 1;

-- name: GetSubscriptionById :one
SELECT s."Id", s."UserId", u."Email" AS "UserEmail", s."PlanId", s."StripeCustomer_id", s."StripeSubscription_id", s."StripeCheckoutSession_id", s."StripePrice_id", s."Status", s."CurrentPeriodStart", s."CurrentPeriodEnd", s."CancelAtPeriodEnd", s."CanceledAt", s."CreatedAt", s."UpdatedAt", p."Name" AS "PlanName", r."Name" AS "PlanRoleName"
FROM "Subscriptions" s
LEFT JOIN "Plans" p ON p."Id" = s."PlanId"
LEFT JOIN "Roles" r ON r."Id" = p."RoleId"
LEFT JOIN "AspNetUsers" u ON u."Id" = s."UserId"
WHERE s."Id" = $1;

-- name: GetSubscriptionByStripeSubscriptionId :one
SELECT "Id", "UserId", "PlanId", "StripeCustomer_id", "StripeSubscription_id", "StripeCheckoutSession_id", "StripePrice_id", "Status", "CurrentPeriodStart", "CurrentPeriodEnd", "CancelAtPeriodEnd", "CanceledAt", "CreatedAt", "UpdatedAt"
FROM "Subscriptions"
WHERE "StripeSubscription_id" = $1;

-- name: UpsertStripeCustomer :exec
INSERT INTO "StripeCustomers" ("UserId", "StripeCustomerId", "CreatedAt", "UpdatedAt")
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT ("UserId") DO UPDATE SET
  "StripeCustomerId" = EXCLUDED."StripeCustomerId",
  "UpdatedAt" = NOW();

-- name: GetStripeCustomerByUserId :one
SELECT "UserId", "StripeCustomerId", "CreatedAt", "UpdatedAt"
FROM "StripeCustomers"
WHERE "UserId" = $1;

-- name: GetUserIdByStripeCustomerId :one
SELECT "UserId"
FROM "StripeCustomers"
WHERE "StripeCustomerId" = $1;

-- name: TryInsertWebhookEvent :one
INSERT INTO "StripeWebhookEvents" ("Id", "Type", "ProcessedAt")
VALUES ($1, $2, NOW())
ON CONFLICT ("Id") DO NOTHING
RETURNING "Id";

-- name: UpsertStripeInvoice :exec
INSERT INTO "StripeInvoices" (
  "StripeInvoiceId", "StripeSubscriptionId", "StripeCustomerId",
  "AmountPaid", "Currency", "Status", "PaidAt", "CreatedAt", "UpdatedAt"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
ON CONFLICT ("StripeInvoiceId") DO UPDATE SET
  "StripeSubscriptionId" = COALESCE(EXCLUDED."StripeSubscriptionId", "StripeInvoices"."StripeSubscriptionId"),
  "StripeCustomerId" = COALESCE(EXCLUDED."StripeCustomerId", "StripeInvoices"."StripeCustomerId"),
  "AmountPaid" = COALESCE(EXCLUDED."AmountPaid", "StripeInvoices"."AmountPaid"),
  "Currency" = COALESCE(EXCLUDED."Currency", "StripeInvoices"."Currency"),
  "Status" = EXCLUDED."Status",
  "PaidAt" = COALESCE(EXCLUDED."PaidAt", "StripeInvoices"."PaidAt"),
  "UpdatedAt" = NOW();

-- name: GetInvoicesByStripeSubscriptionId :many
SELECT "StripeInvoiceId", "AmountPaid", "Currency", "Status", "PaidAt", "CreatedAt"
FROM "StripeInvoices"
WHERE "StripeSubscriptionId" = $1
ORDER BY "CreatedAt" DESC
LIMIT 20;

-- name: GetPlanRoleId :one
SELECT "RoleId"
FROM "Plans"
WHERE "Id" = $1;

-- name: CreatePlan :exec
INSERT INTO "Plans" ("Name", "Description", "Price", "StripePrice_id", "StripeProduct_id", "RoleId", "Active")
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: UpdatePlan :exec
UPDATE "Plans"
SET "Name" = $2, "Description" = $3, "Price" = $4, "StripePrice_id" = $5, "UpdatedAt" = $6, "Active" = $7
WHERE "StripeProduct_id" = $1;

-- name: DeactivatePlan :exec
UPDATE "Plans"
SET "Active" = false, "UpdatedAt" = $2
WHERE "StripeProduct_id" = $1;

-- name: GetPlanByPriceId :one
SELECT "Id", "Name", "Description", "Price", "StripePrice_id", "StripeProduct_id", "Active", "CreatedAt", "UpdatedAt"
FROM "Plans"
WHERE "StripePrice_id" = $1;

-- name: GetPlanByStripeProductID :one
SELECT "Id", "Name", "Description", "Price", "StripePrice_id", "StripeProduct_id", "Active", "CreatedAt", "UpdatedAt"
FROM "Plans"
WHERE "StripeProduct_id" = $1;

-- name: GetPlanById :one
SELECT "Id", "Name", "Description", "Price", "StripePrice_id", "RoleId", "Active", "CreatedAt", "UpdatedAt"
FROM "Plans"
WHERE "Id" = $1;

-- name: GetPlans :many
SELECT "Id", "Name", "Description", "Price"
FROM "Plans"
WHERE "Active" = true
ORDER BY "Price" ASC;

-- name: ExistsPlanByProductId :one
SELECT EXISTS(
    SELECT 1
    FROM "Plans"
    WHERE "StripeProduct_id" = $1
) AS "exists";

-- name: GetRoleByPlan :one
SELECT r."Id"
FROM "Plans" p
JOIN "Roles" r ON r."Id" = p."RoleId"
WHERE p."StripePrice_id" = $1;

-- name: GetAllPlans :many
SELECT "Id", "Name", "Description", "Price", "StripePrice_id", "StripeProduct_id", "RoleId", "Active", "CreatedAt", "UpdatedAt"
FROM "Plans"
ORDER BY "CreatedAt" DESC;

-- name: GetSubscriptions :many
SELECT s."Id", u."Email" AS "UserEmail", s."PlanId", s."StripeCustomer_id", s."StripeSubscription_id", s."StripeCheckoutSession_id", s."StripePrice_id", s."Status", s."CurrentPeriodStart", s."CurrentPeriodEnd", s."CancelAtPeriodEnd", s."CanceledAt", s."CreatedAt", s."UpdatedAt", p."Name" AS "PlanName", r."Name" AS "PlanRoleName"
FROM "Subscriptions" s
LEFT JOIN "Plans" p ON p."Id" = s."PlanId"
LEFT JOIN "AspNetUsers" u ON u."Id" = s."UserId"
LEFT JOIN "Roles" r ON r."Id" = p."RoleId"
ORDER BY s."CreatedAt" DESC;

-- name: GetSubscriptionsByUserId :many
SELECT s."Id", u."Email" AS "UserEmail", s."PlanId", s."StripeCustomer_id", s."StripeSubscription_id", s."StripeCheckoutSession_id", s."StripePrice_id", s."Status", s."CurrentPeriodStart", s."CurrentPeriodEnd", s."CancelAtPeriodEnd", s."CanceledAt", s."CreatedAt", s."UpdatedAt", p."Name" AS "PlanName", r."Name" AS "PlanRoleName"
FROM "Subscriptions" s
LEFT JOIN "Plans" p ON p."Id" = s."PlanId"
LEFT JOIN "AspNetUsers" u ON u."Id" = s."UserId"
LEFT JOIN "Roles" r ON r."Id" = p."RoleId"
WHERE s."UserId" = $1
ORDER BY s."CreatedAt" DESC;

-- name: SaveCvAnalysis :one
INSERT INTO "CvAnalyses" ("UserId", "Score", "JobTitle", "Result", "Active", "CreatedBy", "CreatedAt", "LastModifiedBy", "LastModifiedAt")
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING "Id", "CreatedAt";

-- name: GetCvAnalysesByUser :many
SELECT "Id", "Score", "JobTitle", "Result", "CreatedAt"
FROM "CvAnalyses"
WHERE "UserId" = $1 AND "Active" = true
ORDER BY "CreatedAt" DESC
LIMIT 50;

-- name: GetCvAnalysisById :one
SELECT "Id", "UserId", "Score", "JobTitle", "Result", "CreatedAt"
FROM "CvAnalyses"
WHERE "Id" = $1 AND "UserId" = $2 AND "Active" = true;

-- name: CountUserCvAnalysesSince :one
SELECT COUNT(*)::int FROM "CvAnalyses" WHERE "UserId" = $1 AND "CreatedAt" >= $2;