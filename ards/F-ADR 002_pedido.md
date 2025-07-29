### **F-ADR 002: Ciclo de Vida e Regras de Negócio da Entidade Pedido**

**Status:** Proposto

**Entidade:** Pedido

**Contexto de Negócio:**

O Pedido representa a formalização de uma compra bem-sucedida. Diferente da Reserva, que é um artefato volátil e temporário, o Pedido é um **registro histórico e imutável**. Ele serve como a prova de que uma transação financeira ocorreu, que o estoque foi permanentemente consumido e que a empresa agora tem a obrigação de entregar os ingressos ao cliente. A integridade e a imutabilidade do Pedido são cruciais para a reconciliação financeira, auditoria e para a confiança do cliente no sistema.

#### **1\. Ciclo de Vida da Entidade (State Machine)**

O ciclo de vida do Pedido é deliberadamente simples, pois ele é criado em um estado já quase terminal. Sua principal função é registrar um fato que já ocorreu.

(Reserva Convertida) \---\> \+-----------+        \+-----------------+  
                           |  CRIADO   |-------\>|   FINALIZADO    |  
                           \+-----------+        | (Ingresso Entregue) |  
                                |               \+-----------------+  
                                |  
                                \+--\> (Falha na Entrega) \--\> \+----------------+  
                                                           |   FALHA\_ENTREGA  |  
                                                           \+----------------+

* **Estados:**  
  * **CRIADO**: O estado inicial. O pedido foi criado no banco de dados imediatamente após a confirmação do pagamento (Reserva foi CONVERTIDA). O sistema agora tem a obrigação de gerar e entregar os artefatos (QR Code, e-mail).  
  * **FINALIZADO**: Estado terminal principal. O Componente de Notificações confirmou que o e-mail de confirmação foi enviado e o QR Code foi gerado e associado ao pedido. O ciclo de vida está completo do ponto de vista do sistema.  
  * **FALHA\_ENTREGA**: Estado de exceção. Ocorreu uma falha persistente ao tentar gerar ou enviar o ingresso (ex: o serviço de e-mail ficou indisponível por um longo período). Este estado sinaliza que uma intervenção manual ou um reprocessamento é necessário. O cliente pagou, mas não recebeu o produto.

#### **2\. Atributos Chave**

* id (UUID): Identificador único do pedido.  
* reservaId (UUID, Foreign Key): Link para a Reserva original que deu origem a este pedido. Garante a rastreabilidade.  
* usuarioId (UUID): O usuário que realizou a compra.  
* status (Enum): O estado atual no ciclo de vida (CRIADO, FINALIZADO, FALHA\_ENTREGA).  
* statusPagamento (String): Snapshot do status final retornado pelo gateway (ex: "APROVADO").  
* dadosGateway (JSONB): O payload completo e imutável retornado pelo gateway no momento da confirmação. Essencial para auditoria e disputas.  
* valorTotal (Numeric): O valor total final pago pelo cliente.  
* itens (Array de Structs, via tabela itens\_pedido): A lista de itens comprados. Crucialmente, cada item **armazena o precoUnitarioMomento**, "congelando" o preço daquele momento e protegendo o registro de futuras alterações de preço nos lotes.

#### **3\. Eventos de Domínio Gerados**

* **PedidoConfirmado**: O evento mais importante publicado por este contexto. Ele é disparado quando o pedido transita para o estado CRIADO e a transação do banco de dados é confirmada. É este evento que desacopla a criação do pedido da entrega do ingresso.

#### **4\. Regras de Negócio e Validações**

O Pedido não é criado diretamente por uma ação do usuário, mas sim como um resultado de um processo interno. As validações são, portanto, focadas na consistência do sistema.

**Cenário: Criação de um Pedido (Handler do evento PagamentoProcessado)**

* **QUANDO** o sistema tenta criar um Pedido,  
* **SE** a reservaId correspondente não existir ou seu status não for ATIVA,  
* **ENTÃO** a operação deve falhar e um alerta crítico deve ser gerado, pois representa uma inconsistência grave no fluxo.  
* **QUANDO** o sistema tenta criar um Pedido,  
* **A operação de criação do Pedido e a atualização do status da Reserva para CONVERTIDA devem ocorrer dentro da mesma transação de banco de dados (ACID).**  
* **SE** qualquer parte desta transação falhar,  
* **ENTÃO** um ROLLBACK completo deve ser executado, e o worker deve tentar reprocessar a mensagem (com backoff), para garantir que a compra não seja perdida.  
* **QUANDO** a transação de criação do Pedido é bem-sucedida (COMMIT),  
* **ENTÃO** o status inicial do Pedido deve ser CRIADO, e o evento PedidoConfirmado deve ser imediatamente publicado na fila.

**Cenário: Finalização de um Pedido (Processo do Componente de Notificações)**

* **QUANDO** o Componente de Notificações processa o evento PedidoConfirmado com sucesso,  
* **ENTÃO** ele deve notificar o Componente de Pagamentos (via um evento EntregaRealizada, por exemplo) para que este possa mudar o status do Pedido para FINALIZADO.  
* **QUANDO** o Componente de Notificações falha repetidamente em entregar o ingresso,  
* **ENTÃO** ele deve publicar um evento FalhaNaEntrega para que o status do Pedido seja alterado para FALHA\_ENTREGA e um alerta operacional seja disparado.

#### **5\. Consequências e Implicações de Design**

* **Imutabilidade é Rei:** A estrutura da tabela pedidos e itens\_pedido deve ser tratada como *append-only* (apenas adição). Nenhuma informação crítica (valor, itens, dadosGateway) deve ser alterada após a criação. Correções devem ser feitas através de novos registros de ajuste, se necessário.  
* **Desacoplamento da Entrega:** A publicação do evento PedidoConfirmado é o que permite que o processo de entrega do ingresso (geração de QR Code, envio de e-mail) seja um processo assíncrono e resiliente. O cliente recebe a confirmação de pagamento na tela, enquanto a entrega ocorre em background, segundos depois.  
* **Fonte da Verdade Financeira:** O Pedido é a fonte da verdade para o faturamento. Todos os relatórios financeiros e reconciliações devem ser baseados nos dados imutáveis contidos nestas tabelas