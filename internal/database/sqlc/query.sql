-- name: GetJobs :many
SELECT 
    title,
    description,
    url,
    isApplied,
    status,
    active,
    platform,
    company
FROM jobs;

-- name: GetUserById :one
SELECT id, name, cpf, email, passwordHash, accessFailedCount, onboardingCompleted
FROM users
WHERE id = $1;