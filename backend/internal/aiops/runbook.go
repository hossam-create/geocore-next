package aiops

import (
	"context"
	"fmt"
)

// RunbookGenerator produces actionable remediation steps using LLM + static fallbacks.
type RunbookGenerator struct {
	llm *LLMClient
}

func NewRunbookGenerator(llm *LLMClient) *RunbookGenerator {
	return &RunbookGenerator{llm: llm}
}

const runbookSystemPrompt = `You are a Senior SRE generating an emergency runbook for a production incident.

CONSTRAINTS (non-negotiable):
- Never suggest destructive actions without tagging them "REQUIRES APPROVAL"
- Never auto-rollback production without explicit human sign-off
- Prefer: scale → cache → degrade → rollback (in that order)
- Be specific: include kubectl commands, SQL queries, Redis CLI, Kafka CLI where relevant
- Max 200 words

Respond in exactly this format:
🚨 IMMEDIATE (do now):
1. [specific action]

⚡ MITIGATION (within 5 min):
2. [specific action]
3. [specific action]

🔧 LONG-TERM FIX:
4. [action]

⚠️ REQUIRES APPROVAL:
- [risky action requiring sign-off]`

// Generate produces a runbook for the given incident + RCA.
// Falls back to static runbooks when LLM is unavailable.
func (g *RunbookGenerator) Generate(ctx context.Context, inc *Incident, rca string) string {
	userPrompt := fmt.Sprintf(`Incident: %s
Severity: %s | Service: %s

Root Cause Analysis:
%s

Available tooling: kubectl, Prometheus, Grafana, Kafka CLI, psql, redis-cli`,
		inc.Title, inc.Severity, inc.Service, rca)

	result, err := g.llm.Generate(ctx, runbookSystemPrompt, userPrompt)
	if err != nil || result == "" || result == "[AI analysis disabled — set OPENAI_API_KEY to enable]" {
		return staticRunbook(inc)
	}
	return result
}

// staticRunbook returns pre-built runbooks per service when LLM is not available.
func staticRunbook(inc *Incident) string {
	switch inc.Service {
	case "api":
		return `🚨 IMMEDIATE (do now):
1. kubectl top pods -n default | sort -k3 -rn | head -10

⚡ MITIGATION (within 5 min):
2. kubectl scale deployment api --replicas=$(kubectl get deploy api -o jsonpath='{.spec.replicas}' && echo +2)
3. POST /remediation/enable — enable degraded mode to serve cached responses

🔧 LONG-TERM FIX:
4. Identify slow endpoints in Grafana (API Latency dashboard) and add caching or query optimization

⚠️ REQUIRES APPROVAL:
- Route traffic to secondary region`

	case "database":
		return `🚨 IMMEDIATE (do now):
1. psql -c "SELECT pid,now()-pg_stat_activity.query_start AS dur,query FROM pg_stat_activity WHERE state='active' ORDER BY 1 DESC LIMIT 10;"

⚡ MITIGATION (within 5 min):
2. psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state='active' AND now()-query_start > interval '30s';"
3. Enable read replica routing via /remediation/enable

🔧 LONG-TERM FIX:
4. Tune connection pool (max_open_conns) and add query timeouts

⚠️ REQUIRES APPROVAL:
- Database restart`

	case "kafka":
		return `🚨 IMMEDIATE (do now):
1. kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKERS --describe --all-groups

⚡ MITIGATION (within 5 min):
2. kubectl scale deployment wallet-service --replicas=+2
3. kubectl scale deployment notification-service --replicas=+2
4. kafka-console-consumer.sh --topic orders.events.dlq --from-beginning --max-messages 5

🔧 LONG-TERM FIX:
4. Review consumer throughput — reduce batch size or increase partition count

⚠️ REQUIRES APPROVAL:
- Consumer offset reset`

	case "redis":
		return `🚨 IMMEDIATE (do now):
1. redis-cli INFO memory | grep used_memory_human
2. redis-cli INFO keyspace

⚡ MITIGATION (within 5 min):
2. redis-cli CONFIG SET maxmemory-policy allkeys-lru
3. redis-cli MEMORY DOCTOR

🔧 LONG-TERM FIX:
4. Review TTLs on large keyspaces, add eviction policy

⚠️ REQUIRES APPROVAL:
- FLUSHDB on non-critical keyspace`

	case "wallet":
		return `🚨 IMMEDIATE (do now):
1. STOP all wallet operations — POST /admin/features/wallet_operations/disable
2. Alert finance team immediately

⚡ MITIGATION (within 5 min):
2. Run reconciliation: GET /api/v1/admin/wallet/reconcile
3. Check recent wallet transactions in DB for inconsistencies

🔧 LONG-TERM FIX:
4. Review wallet_invariant_violation_total in Grafana Financial Integrity dashboard

⚠️ REQUIRES APPROVAL:
- Manual ledger correction (requires finance sign-off)`

	default:
		return fmt.Sprintf(`🚨 IMMEDIATE (do now):
1. Check Grafana → System Health dashboard for service: %s
2. kubectl logs -l app=%s --tail=100 | grep -i error

⚡ MITIGATION (within 5 min):
2. kubectl scale deployment %s --replicas=+1
3. Enable degraded mode if user-facing: POST /remediation/enable

🔧 LONG-TERM FIX:
4. Review recent deployments: kubectl rollout history deployment/%s

⚠️ REQUIRES APPROVAL:
- kubectl rollout undo deployment/%s`, inc.Service, inc.Service, inc.Service, inc.Service, inc.Service)
	}
}
