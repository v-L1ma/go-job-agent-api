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

-- name: GetUserById :one
SELECT "Id", "Name", "Cpf", "Email", "PasswordHash", "AccessFailedCount", "OnboardingCompleted"
FROM public."Users"
WHERE "Id" = $1;