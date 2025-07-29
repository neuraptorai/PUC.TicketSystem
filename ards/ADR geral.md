

### **Documento de Arquitetura: Sistema de Venda de Ingressos**

Versão: 1.0  
Data: 29 de Julho de 2025

#### **1\. Visão Geral e Princípios Arquiteturais**

O objetivo deste projeto é construir um sistema de venda de ingressos de alta performance, focado na **velocidade da venda e na experiência do usuário**. O sistema deve ser altamente disponível (SLA de 99,9%), resiliente a falhas e capaz de evoluir para acomodar futuras necessidades de negócio.

Os seguintes princípios guiaram todas as nossas decisões:

* **Pragmatismo Acima do Dogma:** A solução mais simples e eficaz para o contexto do negócio foi preferida em detrimento de soluções academicamente puras, mas excessivamente complexas.  
* **Domain-Driven Design (DDD):** O design é centrado no domínio do negócio, com uma clara separação de responsabilidades em Bounded Contexts e uma Linguagem Ubíqua.  
* **Arquitetura Evolutiva:** O sistema é projetado para começar de forma simples e evoluir de maneira controlada, evitando a superengenharia inicial.  
* **Resiliência e Segurança por Design:** Mecanismos de tolerância a falhas e práticas de segurança são pilares fundamentais da arquitetura, não acréscimos posteriores.

---

#### **2\. Modelo de Domínio e Arquitetura de Software**

##### **2.1. Arquitetura Geral: O Monólito Modular (ADR-001)**

Adotamos uma arquitetura de **Monólito Modular**. A aplicação é um único binário Go, mas seu código é rigorosamente separado em pacotes que representam os Bounded Contexts. Esta abordagem oferece a simplicidade operacional de um monólito com a disciplina de baixo acoplamento dos microsserviços, sendo ideal para a fase inicial do projeto e a experiência da equipe.

##### **2.2. Bounded Contexts e Mapa de Contextos**

Identificamos três contextos principais, que se comunicam de forma assíncrona:

* **Vendas (Sales Context) \- Core Domain:** O coração do negócio. Responsável por gerenciar o inventário de ingressos e as reservas. É onde reside a vantagem competitiva.  
* **Pagamentos (Payments Context) \- Supporting Subdomain:** Responsável por processar o pagamento de uma reserva, interagir com o gateway externo e criar o pedido final. É essencial, mas não é o diferencial do negócio.  
* **Notificações (Notifications Context) \- Generic Subdomain:** Responsável pelo envio de comunicações (e-mails, QR Codes, lembretes). É um problema genérico, resolvido com ferramentas de mercado.

O mapa de comunicação entre eles é o seguinte:

\[Vendas\] \-- (publica ReservaCriada, ReservaExpirada) \--\> \[Fila de Mensagens\]  
\[Pagamentos\] \-- (publica PedidoConfirmado) \--\> \[Fila de Mensagens\]  
\[Fila de Mensagens\] \--\> (consomem eventos) \[Pagamentos\], \[Notificações\]

##### **2.3. Padrões Arquiteturais Críticos**

* **CQRS para Gestão de Estoque (ADR-002):** Para garantir performance extrema, separamos os caminhos de leitura e escrita do estoque. Reservas (comandos) são operações atômicas em um cache **Redis**. Consultas de disponibilidade (queries) leem diretamente deste cache. O banco de dados relacional é atualizado de forma assíncrona, eliminando o gargalo de contenção.  
* **Comunicação Assíncrona via Fila de Mensagens (ADR-003):** Toda a comunicação entre contextos é assíncrona, mediada por uma fila (ex: RabbitMQ). Isso garante resiliência e isolamento de falhas. Uma lentidão no contexto de Pagamentos não afeta a capacidade de criar novas reservas no contexto de Vendas.  
* **Arquitetura Reativa no Contexto de Pagamentos (ADR-005):** A lógica de negócio no contexto de pagamentos é encapsulada em *workers* que reagem a eventos da fila. Isso torna o processo resiliente e alinhado à natureza assíncrona do fluxo de pagamento.

---

#### **3\. Contratos de Dados e APIs**

##### **3.1. Schema do Banco de Dados (PostgreSQL \- A Fonte da Verdade)**

* eventos: Armazena os eventos principais.  
* tipos\_ingresso: Define os tipos de ingressos, lotes, preços e o **estoque inicial**.  
* reservas e itens\_reserva: Rastreiam as reservas temporárias, seu status e tempo de expiração.  
* pedidos e itens\_pedido: Armazenam os pedidos confirmados de forma imutável, incluindo um snapshot do preço no momento da compra.

##### **3.2. Estrutura do Cache de Estoque (Redis)**

* **Tipo:** Redis Hash  
* **Chave:** estoque:evento:{id\_do\_evento}  
* **Campos do Hash:**  
  * **Chave do campo:** {id\_tipo\_ingresso}  
  * **Valor:** \<contador\_atual\_de\_ingressos\_disponíveis\>  
* **Operação de Reserva:** HINCRBY estoque:evento:{id} {id\_tipo\_ingresso} \-{quantidade}

##### **3.3. Contratos de Eventos (JSON \- Linguagem Publicada)**

* **ReservaCriada**:  
  JSON  
  {  
    "eventId": "ReservaCriada", "eventVersion": "1.0",  
    "payload": {  
      "reservationId": "uuid", "userId": "uuid", "expiresAt": "timestamp",  
      "items": \[{ "ticketTypeId": "uuid", "quantity": 2, "unitPrice": "250.00" }\],  
      "totalAmount": "500.00"  
    }  
  }

* **PedidoConfirmado**:  
  JSON  
  {  
    "eventId": "PedidoConfirmado", "eventVersion": "1.0",  
    "payload": {  
      "orderId": "uuid", "userId": "uuid", "userEmail": "cliente@email.com",  
      "items": \[{ "ticketTypeName": "VIP", "lot": "Lote 1", "quantity": 2 }\]  
    }  
  }

##### **3.4. Camada Anticorrupção (ACL) para Pagamentos (ADR-004)**

A integração com o Mercado Pago é isolada através do padrão **Adapter**, implementando uma interface genérica PaymentGateway para proteger nosso domínio dos detalhes do sistema externo.

Go

// A interface que nosso domínio conhece  
type PaymentGateway interface {  
    CriarIntencaoDePagamento(ctx context.Context, reserva Reserva) (IntencaoPagamento, error)  
    ConsultarStatusPagamento(ctx context.Context, idPagamentoExterno string) (StatusPagamento, error)  
}

##### **3.5. API Pública (RESTful com JWT \- ADR-006)**

A fachada do sistema é uma API RESTful versionada e segura.

* **Autenticação:** Via JSON Web Tokens (JWT) para endpoints que exigem contexto de usuário.  
* **Endpoints Principais:**  
  * GET /api/v1/eventos/{id}/disponibilidade: Consulta de estoque (CQRS) lendo do Redis e do PostgreSQL.  
  * POST /api/v1/reservas: Cria uma reserva temporária, retornando 202 Accepted para indicar processamento assíncrono.  
  * GET /api/v1/reservas/{id}: Endpoint de polling para o front-end buscar os dados de pagamento (QR Code) após a reserva.  
  * GET /api/v1/pedidos: Histórico de pedidos do usuário autenticado.

---

#### **4\. Infraestrutura, Operações e Ciclo de Vida do Desenvolvimento**

##### **4.1. Infraestrutura como Serviço Gerenciado (ADR-007)**

Para atingir o SLA e focar no desenvolvimento, toda a infraestrutura será consumida como serviço gerenciado em nuvem.

* **Orquestração:** **Docker** para conteinerização e um serviço gerenciado de **Kubernetes** (GKE, EKS, AKS) para orquestração, escalabilidade automática e autocorreção.  
* **Serviços Stateful:** Serviços gerenciados de alta disponibilidade para **PostgreSQL** (ex: RDS), **Redis** (ex: ElastiCache) e para a **Fila de Mensagens**.

##### **4.2. Estratégia de Deployment: Blue-Green**

Para garantir deploys com zero downtime, adotaremos a estratégia **Blue-Green**. Uma nova versão da aplicação é implantada em um ambiente "Green" idêntico ao de produção ("Blue"). Após a validação, o tráfego é instantaneamente direcionado para o novo ambiente, permitindo rollback imediato em caso de problemas.

##### **4.3. Observabilidade (Os Três Pilares)**

* **Logs Estruturados:** Logs em formato JSON centralizados em uma plataforma de análise (Loki, Elasticsearch).  
* **Métricas:** Métricas de negócio e de sistema exportadas no formato Prometheus e visualizadas no Grafana, com alertas proativos.  
* **Traces (Rastreamento):** Implementação de OpenTelemetry para rastrear o ciclo de vida de uma requisição através dos componentes assíncronos, essencial para depuração.

##### **4.4. Esteira de CI/CD Automatizada (ADR-008)**

O ciclo de vida do desenvolvimento será governado por uma esteira automatizada (ex: GitHub Actions).

* **Integração Contínua (CI):** A cada push, a esteira executa:  
  1. Build do código.  
  2. Análise estática e linting para garantir padrões arquiteturais.  
  3. Testes unitários e de integração (contra instâncias Docker de BD e Redis).  
  4. Análise de segurança.  
  5. Construção e publicação da imagem Docker.  
* **Deploy Contínuo (CD):** Após o merge para a main:  
  1. Deploy automático no ambiente de **Staging**.  
  2. Execução de testes End-to-End.  
  3. **Aprovação manual** para o deploy em produção.  
  4. Execução do deploy Blue-Green em produção.

---

Este documento representa o estado atual do nosso design arquitetural. É um guia vivo, destinado a orientar o desenvolvimento, a operação e a evolução contínua do sistema de venda de ingressos.