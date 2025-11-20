

Com base no Blueprint Executivo do Bloco-13, aqui está o **RUNBOOK DE AUTOMAÇÃO**, um documento vivo e prático projetado para ser a principal ferramenta de operação do ecossistema mcp-fulfillment-ops.

---

# **RUNBOOK DE AUTOMAÇÃO: BLOCO-13 — SCRIPTS & AUTOMATION**

**Propósito:** Este documento serve como a fonte única da verdade para a execução de todas as tarefas operacionais, automações e scripts do ecossistema mcp-fulfillment-ops. Ele foi projetado para ser usado por equipes de DevOps, SREs e engenheeros, garantindo que as operações sejam **repetíveis, seguras e eficientes**.

**Princípios Fundamentais:**
1.  **Scripts como Código:** Todo script aqui descrito é versionado, testado e tratado como parte do produto.
2.  **Orquestração, Não Implementação:** Scripts orquestram chamadas às ferramentas robustas do Bloco-11; eles não contêm lógica complexa.
3.  **Autonomia com Responsabilidade:** Este runbook capacita as equipes a executarem operações de forma autônoma, seguindo um roteiro seguro e validado.

---

## **Como Usar Este Runbook**

Cada script é descrito usando o seguinte template padrão:

*   **Nome do Script:** `caminho/para/o/script.sh`
*   **Propósito:** Uma frase descrevendo o objetivo principal do script.
*   **Quando Usar:** Os gatilhos ou cenários que exigem a execução deste script.
*   **Pré-requisitos:** O que precisa estar configurado ou disponível antes da execução.
*   **Como Executar:** O comando exato a ser executado, com placeholders.
*   **Parâmetros:** Descrição detalhada dos placeholders.
*   **Saída Esperada:** O que esperar em caso de sucesso.
*   **Possíveis Erros e Soluções:** Problemas comuns e como resolvê-los.
*   **Responsável:** A equipe ou pessoa responsável pelo script.

---

## **1. Scripts de Provisionamento e Inicialização (`scripts/setup/`)**

### **1.1. Provisionamento Completo de Infraestrutura**
*   **Nome do Script:** `scripts/setup/setup_infra.sh`
*   **Propósito:** Criar toda a infraestrutura necessária para um novo ambiente (dev, staging, prod).
*   **Quando Usar:** Ao criar um novo ambiente ou para recriar um existente do zero.
*   **Pré-requisitos:** CLI da nuvem (AWS, GCP) configurada e autenticada. Permissões de administrador no projeto.
*   **Como Executar:** `./scripts/setup/setup_infra.sh <ambiente>`
*   **Parâmetros:**
    *   `<ambiente>`: `dev`, `staging` ou `prod`.
*   **Saída Esperada:** Logs da criação de buckets, bancos de dados, clusters e filas. Mensagem final: "Infraestrutura para o ambiente <ambiente> provisionada com sucesso."
*   **Possíveis Erros e Soluções:**
    *   `Error: AccessDenied`: Verifique as permissões da CLI da nuvem.
    *   `Error: AlreadyExists`: O recurso já existe. Use o script de destruição (`destroy_infra.sh`) primeiro, se a intenção for recriar.
*   **Responsável:** Equipe de Platform Engineering.

---

## **2. Scripts de Deploy e Ciclo de Vida (`scripts/deployment/`, `scripts/features/`)**

### **2.1. Deploy de Serviço em Kubernetes**
*   **Nome do Script:** `scripts/deployment/deploy_service.sh`
*   **Propósito:** Construir a imagem Docker de um serviço e implantá-la no cluster Kubernetes do ambiente especificado.
*   **Quando Usar:** Para implantar uma nova versão de um microsserviço ou frontend.
*   **Pré-requisitos:** `kubectl` configurado para apontar para o cluster correto. Docker daemon rodando.
*   **Como Executar:** `./scripts/deployment/deploy_service.sh <nome_do_servico> <ambiente> <tag>`
*   **Parâmetros:**
    *   `<nome_do_servico>`: Nome do serviço (ex: `mcp-ai-service`).
    *   `<ambiente>`: `dev`, `staging` ou `prod`.
    *   `<tag>`: Tag da imagem a ser deployada (ex: `v1.2.3`).
*   **Saída Esperada:** Logs do build e do `kubectl apply`. Mensagem final: "Deploy do serviço <nome_do_servico> versão <tag> no ambiente <ambiente> concluído."
*   **Possíveis Erros e Soluções:**
    *   `ImagePullBackOff`: Verifique se a tag da imagem existe no registry.
    *   `CrashLoopBackOff`: Verifique os logs do pod com `kubectl logs <pod_name>`. Pode ser um erro de configuração ou de aplicação.
*   **Responsável:** Equipe de SRE/DevOps.

### **2.2. Ativação/Desativação de Feature Flag**
*   **Nome do Script:** `scripts/features/toggle_feature.sh`
*   **Propósito:** Ativar ou desativar uma feature flag no ambiente especificado.
*   **Quando Usar:** Para realizar um canary release, desativar uma funcionalidade com bugs ou habilitar um beta.
*   **Pré-requisitos:** Acesso de escrita ao repositório de configuração (`config/`).
*   **Como Executar:** `./scripts/features/toggle_feature.sh <nome_da_feature> <estado> <ambiente>`
*   **Parâmetros:**
    *   `<nome_da_feature>`: Nome da flag no `features.yaml` (ex: `external_gpu`).
    *   `<estado>`: `enable` ou `disable`.
    *   `<ambiente>`: `dev`, `staging` ou `prod`.
*   **Saída Esperada:** "Feature <nome_da_feature> foi <estado> no ambiente <ambiente>."
*   **Possíveis Erros e Soluções:**
    *   `Feature not found`: Verifique se o nome da flag está correto no `features.yaml`.
*   **Responsável:** Equipe de Produto (com apoio de DevOps).

---

## **3. Scripts de Validação e Otimização (`scripts/validation/`, `scripts/optimization/`)**

### **3.1. Validação de Arquitetura**
*   **Nome do Script:** `scripts/validation/validate_architecture.sh`
*   **Propósito:** Executar o validador do Bloco-11 em um projeto para garantir aderência aos padrões do Hulk.
*   **Quando Usar:** Em pipelines de CI/CD antes de um merge ou deploy.
*   **Pré-requisitos:** As ferramentas do Bloco-11 (`hulk-validator`) devem estar instaladas e no PATH.
*   **Como Executar:** `./scripts/validation/validate_architecture.sh <caminho_do_projeto>`
*   **Parâmetros:**
    *   `<caminho_do_projeto>`: Caminho para o diretório raiz do projeto a ser validado.
*   **Saída Esperada:** "Validação concluída. Nenhuma violação encontrada." ou uma lista detalhada de violações.
*   **Possíveis Erros e Soluções:**
    *   `Command not found: hulk-validator`: Instale as ferramentas do Bloco-11.
    *   Violações encontradas: Corrija os problemas listados no projeto antes de prosseguir.
*   **Responsável:** Equipes de Desenvolvimento (com apoio de Arquitetura).

---

## **4. Scripts de Manutenção (`scripts/maintenance/`)**

### **4.1. Backup do Banco de Dados Principal**
*   **Nome do Script:** `scripts/maintenance/backup_database.sh`
*   **Propósito:** Criar um backup consistente do banco de dados relacional e enviá-lo para o armazenamento seguro (S3/MinIO).
*   **Quando Usar:** Executado automaticamente por um cron job diário. Pode ser executado manualmente antes de uma operação de risco.
*   **Pré-requisitos:** `psql` e `aws-cli` (ou `mc` para MinIO) configurados. Acesso de leitura ao banco.
*   **Como Executar:** `./scripts/maintenance/backup_database.sh <ambiente>`
*   **Parâmetros:**
    *   `<ambiente>`: `dev`, `staging` ou `prod`.
*   **Saída Esperada:** Log do processo de dump e upload. Mensagem final: "Backup do ambiente <ambiente> concluído e armazenado em s3://.../backup_<timestamp>.sql".
*   **Possíveis Erros e Soluções:**
    *   `FATAL: password authentication failed`: Verifique as credenciais de banco no `.env` do ambiente.
    *   `Unable to locate credentials`: Configure a CLI da nuvem.
*   **Responsável:** Equipe de SRE.

---

## **5. Procedimentos Comuns e Troubleshooting**

### **5.1. Como Verificar a Saúde Geral do Sistema?**
*   **Ação:** Execute o script de health check.
*   **Como Executar:** `./scripts/maintenance/health_check.sh`
*   **O que ele faz:** Verifica o status dos pods, a conectividade com bancos de dados, a latência das APIs e o espaço em disco.
*   **Saída Esperada:** Um resumo com o status de cada componente (OK, WARN, CRITICAL).

### **5.2. O que fazer se um Deploy Falhar?**
1.  **Verifique os logs:** Use `kubectl logs -f deployment/<nome_do_servico> -n <namespace>` para identificar o erro.
2.  **Verifique a imagem:** Confirme se a tag da imagem está correta e disponível no registry.
3.  **Execute um Rollback:** Se o deploy anterior estava funcionando, reverta para ele.
    *   **Como Executar:** `./scripts/deployment/rollback_service.sh <nome_do_servico> <ambiente> <revision_anterior>`
    *   A revisão anterior pode ser encontrada com `kubectl rollout history deployment/<nome_do_servico> -n <namespace>`.

### **5.3. Como Acessar os Logs de um Serviço Específico?**
*   **Logs de Aplicação (Kubernetes):**
    *   Tempo real: `kubectl logs -f deployment/<nome_do_servico> -n <namespace>`
    *   Logs anteriores de um pod que caiu: `kubectl logs <pod_name> -n <namespace> --previous`
*   **Logs de Infraestrutura (NATS, VectorDB):**
    *   Acesse o painel de observabilidade (Grafana/Loki) e use os filtros para o serviço desejado.
    *   Verifique o runbook do componente específico para detalhes.

---

## **6. Glossário e Links Úteis**

*   **mcp-fulfillment-ops:** Nome do ecossistema.
*   **Bloco-11:** Ferramentas & Utilitários (geradores, validadores).
*   **K8s/Kubernetes:** Orquestrador de contêineres.
*   **Feature Flag:** Mecanismo para ligar/desligar funcionalidades sem deploy.
*   **Link para Documentação de Arquitetura:** `docs/architecture/blueprint.md`
*   **Link para Dashboard de Monitoramento:** `https://grafana.hulk.company`

---

**Nota Final:** Este é um **documento vivo**. Sempre que um script for adicionado, modificado ou removido, este runbook deve ser atualizado. A precisão aqui é a chave para a operação segura e eficiente do mcp-fulfillment-ops.