# go-job-agent-api
Uma api em golang para o job agent


login ===================== FEITO
cadastro ===================== FEITO
esqueci minha senha
refresh token

evaluateJob ===================== FEITO
getJobById ===================== FEITO
getJobs ===================== FEITO
jobMatchScore - esse de alguma forma precisa ser dentro de getJobs ou por um cron 

evaluateGeneratedCV ===================== FEITO
generateCv ===================== FEITO
SavePreferences ===================== FEITO
uploadCv ===================== FEITO

getGeneratedCvs ===================== FEITO
getPreferences ===================== FEITO
getUserCV ===================== FEITO

getUserProfile ===================== FEITO
updateProfile ===================== FEITO
changePassword ===================== FEITO
getuserStatistics ===================== FEITO

evaluateUserCV - listar pontos de melhorias no curriculo atual do usuário
Nao tenho curriculo - opçao na mesma tela de curriculo que permite o usuario criar um curriculo para ele

applyJob - preciso criar uma rota para marcar quais vagas foram aplicadas para quais usuarios, criar uma tabela com userId, JobId, Status, Obs

getRespostas
getCandidaturas

parece que o search queries nao esta sendo alterado ao settar as users preferences e a tabela de users preferences nem deveria ser mais utilizada

tem q ver se o get de preferences ta trazendo de acordo com a search query