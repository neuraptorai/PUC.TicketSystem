### **F-ADR 001: Ciclo de Vida e Regras de Negócio da Entidade Reserva**

**Status:** Proposto

**Entidade:** Reserva

**Contexto de Negócio:**

A entidade Reserva existe para resolver o problema da concorrência por um recurso escasso (o ingresso). Seu propósito fundamental é **reter temporariamente um ou mais ingressos para um usuário específico**, dando-lhe uma janela de tempo para concluir o pagamento sem o risco de outra pessoa comprar os mesmos ingressos. Ela atua como a ponte transacional entre a intenção de compra do usuário (Vendas Context) e a confirmação financeira (Pagamentos Context). Uma gestão incorreta do ciclo de vida da Reserva leva diretamente a problemas de negócio como overselling ou "estoque fantasma" (ingressos presos que não podem ser vendidos).

#### **1\. Ciclo de Vida da Entidade (State Machine)**

A Reserva opera como uma máquina de estados finitos. Uma vez que entra em um estado terminal (CONVERTIDA, EXPIRADA, CANCELADA), seu ciclo de vida termina.

                  \+------------------+  
(Início) \-------\> |      ATIVA       | \-------\> (Pagamento Confirmado) \----+  
                  | (Estoque Retido) |                                     |  
                  \+------------------+                                     |  
                         |     |                                          V  
                         |     \+-----\> (Usuário Cancela) \----\> \+-----------+----+  
                         |                                     |   CANCELADA    |  
                         |                                     | (Estoque Livre) |  
                         |                                     \+----------------+  
                         |  
                         |  
                         \+-----\> (Tempo Esgotado) \-------\> \+-----------+----+  
                                                           |   EXPIRADA     |  
                                                           | (Estoque Livre)|  
                                                           \+----------------+

                  \+-----------+----+  
                  |  CONVERTIDA    |  
                  | (Ligada a Pedido)|  
                  \+----------------+

* **Estados:**  
  * **ATIVA**: O estado inicial. A reserva foi criada, o estoque foi decrementado no cache (Redis) e o sistema aguarda a ação do usuário (pagamento). A reserva possui um tempo de expiração (expiraEm).  
  * **CONVERTIDA**: Estado terminal. O pagamento foi confirmado com sucesso. A reserva cumpriu seu propósito e foi convertida em um Pedido. O estoque que ela retinha agora está permanentemente alocado.  
  * **EXPIRADA**: Estado terminal. O tempo em expiraEm foi atingido antes da confirmação do pagamento. O estoque retido deve ser liberado.  
  * **CANCELADA**: Estado terminal. O usuário decidiu ativamente cancelar a reserva antes do pagamento. O estoque retido deve ser liberado.

#### **2\. Atributos Chave**

* id (UUID): Identificador único da reserva.  
* usuarioId (UUID): O usuário que detém a reserva.  
* status (Enum): O estado atual no ciclo de vida (ATIVA, CONVERTIDA, EXPIRADA, CANCELADA).  
* expiraEm (Timestamp): Momento exato em que a reserva se tornará EXPIRADA se não for convertida.  
* itens (Array de Structs): A lista de itens reservados, cada um contendo tipoIngressoId e quantidade.  
* dadosGateway (JSONB, Nulável): Armazena os dados retornados pelo gateway de pagamento (ex: QR Code PIX, link de pagamento) necessários para o front-end.

#### **3\. Eventos de Domínio Gerados**

* **ReservaCriada**: Publicado quando a reserva transita para o estado ATIVA.  
* **ReservaExpirada**: Publicado quando a reserva transita para o estado EXPIRADA.  
* **ReservaCancelada**: Publicado quando a reserva transita para o estado CANCELADA.

#### **4\. Regras de Negócio e Validações**

Estas são as invariantes que o sistema deve garantir em todos os momentos.

**Cenário: Criação de uma Reserva (POST /api/v1/reservas)**

* **QUANDO** um usuário tenta criar uma reserva,  
* **SE** o usuário não estiver autenticado,  
* **ENTÃO** a operação deve ser rejeitada com um erro 401 Unauthorized.  
* **QUANDO** um usuário tenta criar uma reserva,  
* **SE** os tipos\_ingresso solicitados não pertencem a um evento ativo ou não existem,  
* **ENTÃO** a operação deve ser rejeitada com um erro 400 Bad Request.  
* **QUANDO** um usuário tenta criar uma reserva,  
* **SE** a quantidade solicitada para qualquer tipo\_ingresso for maior que o estoque disponível no cache (Redis),  
* **ENTÃO** a operação deve ser rejeitada com um erro 409 Conflict (conflito de estoque).  
* **QUANDO** um usuário tenta criar uma reserva,  
* **SE** este mesmo usuário já possui uma outra reserva ATIVA,  
* **ENTÃO** a nova operação deve ser rejeitada com um erro 409 Conflict (reserva ativa existente), para evitar que um usuário "segure" grande parte do estoque.  
* **QUANDO** a criação da reserva é bem-sucedida,  
* **ENTÃO** o status deve ser ATIVA, o expiraEm deve ser calculado (agora() \+ 10 minutos), e o evento ReservaCriada deve ser publicado.

**Cenário: Expiração de uma Reserva (Processo de Worker)**

* **QUANDO** o processo de expiração é executado,  
* **SE** ele encontrar uma reserva com status \= ATIVA e expiraEm \< agora(),  
* **ENTÃO** a transição para o estado EXPIRADA deve ser atômica e garantir duas ações:  
  1. A quantidade de cada item da reserva deve ser **devolvida** ao estoque correspondente no cache (Redis), usando uma operação HINCRBY.  
  2. O status da reserva no banco de dados deve ser alterado para EXPIRADA.  
  3. O evento ReservaExpirada deve ser publicado.

**Cenário: Pagamento de uma Reserva**

* **QUANDO** o sistema recebe uma confirmação de pagamento para uma Reserva,  
* **SE** o status da reserva já for EXPIRADA ou CANCELADA,  
* **ENTÃO** a operação deve ser rejeitada e, idealmente, o sistema deve iniciar um processo de reembolso (se o dinheiro foi capturado), logando um alerta crítico.  
* **QUANDO** o sistema recebe uma confirmação de pagamento para uma Reserva,  
* **SE** o status da reserva for ATIVA,  
* **ENTÃO** o sistema deve transicionar o status para CONVERTIDA e criar um Pedido correspondente. A reserva não pode mais expirar.

#### **5\. Consequências e Implicações de Design**

* **Worker de Expiração é Crítico:** O processo que verifica e expira as reservas é uma peça fundamental da infraestrutura. Sua falha ou atraso pode levar ao bloqueio indefinido do estoque. Ele deve ser altamente monitorado.  
* **Atomicidade da Liberação de Estoque:** A operação de mudar o status para EXPIRADA/CANCELADA e devolver o estoque ao Redis deve ser tratada com cuidado para evitar inconsistências. Uma abordagem de "melhor esforço" com retries e alertas é necessária.  
* **Interface do Usuário:** O front-end deve ser notificado claramente sobre o tempo de expiração da reserva, com um contador regressivo, para criar um senso de urgência e gerenciar as expectativas do usuário.

