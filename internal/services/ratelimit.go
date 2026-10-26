package services

// Rate limit simples em memória, sem dependência externa.
//
// Usado pela rota pública de análise de currículo (POST /cv-analysis/public)
// para limitar chamadas por IP. Implementa janela deslizante: mantém os
// timestamps das tentativas dentro da janela e os descarta a cada chamada,
// então o mapa só cresce enquanto houver chaves ativas.

import (
	"sync"
	"time"
)

// RateLimiter limita "limit" chamadas de Allow por chave dentro de "window".
// Seguro para uso concorrente (mutex único por instância).
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string][]time.Time
}

// NewRateLimiter cria um limitador de "limit" eventos por "window" por chave.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
		hits:   make(map[string][]time.Time),
	}
}

// Allow registra uma tentativa da chave e devolve false quando o limite da
// janela deslizante já foi atingido (a tentativa não é contada quando
// rejeitada).
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Varredura de limpeza: descarta tentativas fora da janela e, quando a
	// chave fica sem nenhuma tentativa válida, remove a entrada do mapa.
	for k, times := range rl.hits {
		kept := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(rl.hits, k)
			continue
		}
		rl.hits[k] = kept
	}

	if len(rl.hits[key]) >= rl.limit {
		return false
	}

	rl.hits[key] = append(rl.hits[key], now)
	return true
}
