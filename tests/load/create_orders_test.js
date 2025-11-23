import http from 'k6/http';
import { check, sleep } from 'k6';

// Configurações do teste
const BASE_URL = 'http://localhost:8082';
const CONCURRENT_USERS = 100;
const TEST_DURATION = '30s';
const RAMP_UP_TIME = '10s';

// Métricas personalizadas
let ordersCreated = 0;
let ordersFailed = 0;
let responseTimes = [];

export let options = {
  stages: [
    { duration: RAMP_UP_TIME, target: CONCURRENT_USERS },
    { duration: TEST_DURATION, target: CONCURRENT_USERS },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% das requisições < 500ms
    http_req_failed: ['rate<0.1'], // Taxa de falha < 0.1%
  },
};

export default function () {
  // Grupo de requisições para criação de ordens
  const createOrderGroup = http.group('Create Orders', {
    minIteration: 100, // Coletar estatísticas após 100 iterações
  });

  // Teste de criação de ordens
  createOrderGroup.post(`${BASE_URL}/api/v1/fulfillment-orders`, {
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      order_id: `ORDER-${__VU}-${__ITER}`,
      customer: `CUSTOMER-${__VU}`,
      destination: 'Rua Teste K6, 123 - São Paulo/SP',
      items: [
        { sku: 'PROD-K6-001', quantity: Math.floor(Math.random() * 5) + 1 },
      ],
      priority: 0,
      idempotency_key: `KEY-${__VU}-${__ITER}`,
    }),
  });

  // Medir tempo de resposta
  createOrderGroup.on('response', (res) => {
    if (res.status === 201 || res.status === 200) {
      ordersCreated++;
      responseTimes.push(res.timings.duration);
    } else {
      ordersFailed++;
    }
  });

  // Teste de consulta de ordens
  const getOrderGroup = http.group('Get Orders', {
    minIteration: 50,
  });

  getOrderGroup.get(`${BASE_URL}/api/v1/fulfillment-orders/ORDER-${__VU}-1`, {
    headers: {
      'Content-Type': 'application/json',
    },
  });

  // Teste de health check
  const healthGroup = http.group('Health Check', {
    minIteration: 10,
  });

  healthGroup.get(`${BASE_URL}/health`);

  // Teste de métricas
  const metricsGroup = http.group('Metrics', {
    minIteration: 20,
  });

  metricsGroup.get(`${BASE_URL}/metrics`);

  // Relatório final
  createOrderGroup.on('end', () => {
    const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
    const maxResponseTime = Math.max(...responseTimes);
    const p95ResponseTime = responseTimes.sort((a, b) => a - b)[Math.floor(responseTimes.length * 0.95)];

    console.log('\n=== RELATÓRIO DE TESTE DE CARGA ===');
    console.log(`Ordens criadas: ${ordersCreated}`);
    console.log(`Ordens falharam: ${ordersFailed}`);
    console.log(`Taxa de sucesso: ${((ordersCreated / (ordersCreated + ordersFailed)) * 100).toFixed(2)}%`);
    console.log(`Tempo médio de resposta: ${avgResponseTime.toFixed(2)}ms`);
    console.log(`Tempo máximo de resposta: ${maxResponseTime}ms`);
    console.log(`P95 de resposta: ${p95ResponseTime}ms`);
    console.log(`Requisições por segundo: ${(ordersCreated / TEST_DURATION.replace('s', '')).toFixed(2)}`);
  });

  sleep(1);
}