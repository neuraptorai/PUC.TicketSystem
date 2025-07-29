### **F-ADR 003: Processo de Negócio para Devolução de Estoque**

**Status:** Proposto

**Processo:** Devolução de Estoque

**Contexto de Negócio:**

Este processo governa a lógica pela qual os ingressos, temporariamente retidos por uma Reserva, são devolvidos ao pool de estoque disponível. Ele é a contraparte direta e necessária do processo de criação de reserva. Sem uma devolução de estoque robusta e confiável, o sistema sofreria de "estoque fantasma" (ou *ghost inventory*): ingressos que aparecem como indisponíveis, mas que na realidade não foram vendidos e não podem mais ser comprados. A automação e a exatidão deste processo são vitais para maximizar a receita e garantir a justiça no acesso aos ingressos pelos usuários.

#### **1\. Gatilhos do Processo (Triggers)**

A devolução de estoque é iniciada por dois eventos de negócio distintos:

1. **Expiração da Reserva:** Um evento sistêmico, automático. Ocorre quando o tempo de vida de uma Reserva ATIVA (definido pelo seu atributo expiraEm) se esgota sem que um pagamento tenha sido confirmado.  
2. **Cancelamento pelo Usuário:** Um evento explícito, iniciado pelo usuário. Ocorre quando o detentor de uma Reserva ATIVA decide voluntariamente abandonar sua intenção de compra através de uma ação na interface.

#### **2\. Atores e Implementação Técnica**

* **Para a Expiração da Reserva (Ator: Worker de Expiração):**  
  * Um processo em background (um *worker* ou *cron job*), rodando em intervalos regulares (ex: a cada minuto).  
  * **Lógica:**  
    1. Escaneia a tabela reservas no banco de dados em busca de registros onde: status \= 'ATIVA' E expiraEm \<= now().  
    2. Para cada reserva encontrada, ele inicia o processo de devolução.  
* **Para o Cancelamento pelo Usuário (Ator: API Server):**  
  * Um endpoint específico na nossa API: DELETE /api/v1/reservas/{reservaId}.  
  * **Lógica:**  
    1. O endpoint recebe a requisição de cancelamento.  
    2. Verifica se o usuário autenticado é de fato o "dono" da reservaId informada.  
    3. Inicia o processo de devolução.

#### **3\. Lógica Central do Processo (As Ações)**

Independentemente do gatilho, a lógica de negócio executada é a mesma e deve ser tratada como uma operação atômica ou, mais pragmaticamente, como uma sequência de passos que deve ser resiliente a falhas.

1. **Liberação do Estoque no Cache:** Para cada item na Reserva, o sistema deve executar uma operação atômica de incremento no Redis: HINCRBY estoque:evento:{id} {id\_tipo\_ingresso} \+{quantidade}. Esta é a ação mais crítica, pois libera o ingresso para outros usuários imediatamente.  
2. **Atualização do Estado da Entidade:** O status da Reserva no banco de dados PostgreSQL deve ser alterado para o estado terminal apropriado: EXPIRADA ou CANCELADA. Isso remove a reserva do conjunto de candidatas a futuras expirações ou pagamentos.  
3. **Publicação do Evento de Domínio:** Após a conclusão bem-sucedida dos passos anteriores, o sistema deve publicar o evento correspondente na fila: ReservaExpirada ou ReservaCancelada. Este evento pode ser consumido por outros componentes, como o de Notificações (para enviar um e-mail "Você perdeu seus ingressos...") ou um módulo de análise.

#### **4\. Regras de Negócio e Validações**

* **QUANDO** o processo de devolução de estoque é acionado para uma Reserva,  
* **SE** o status atual da Reserva não for ATIVA,  
* **ENTÃO** a operação deve ser interrompida. Isso previne que o estoque de uma reserva já CONVERTIDA, EXPIRADA ou CANCELADA seja devolvido indevidamente (o que corromperia o estoque total).  
* **QUANDO** o processo de devolução de estoque é acionado,  
* **A operação deve ser idempotente.**  
* **SE** o mesmo processo tentar rodar duas vezes para a mesma reservaId,  
* **ENTÃO** apenas a primeira execução deve ter efeito. A segunda tentativa, ao verificar que o status não é mais ATIVA, falhará a validação e não fará nada, garantindo a integridade do estoque.  
* **QUANDO** um usuário tenta cancelar uma reserva (DELETE /api/v1/reservas/{reservaId}),  
* **SE** o usuarioId do token JWT não for o mesmo usuarioId associado à reserva,  
* **ENTÃO** a operação deve ser rejeitada com um erro 403 Forbidden.

#### **5\. Consequências e Implicações de Design**

* **Alta Disponibilidade do Worker de Expiração:** O worker que executa a expiração é um componente crítico da infraestrutura de negócio. Ele não pode falhar. Deve ser monitorado de perto, com alertas para qualquer interrupção em sua execução.  
* **Risco de Inconsistência (Cache vs. BD):** Existe um risco teórico de que o estoque seja devolvido no Redis, mas a atualização do status no PostgreSQL falhe. O processo deve ser construído com uma lógica de *retry*. Se a falha persistir, um alerta crítico deve ser gerado. A prioridade é sempre liberar o estoque no cache para maximizar a oportunidade de venda.  
* **Não Reversível:** A transição para os estados EXPIRADA ou CANCELADA é final. Não existe um fluxo de negócio para "reativar" uma reserva. O usuário que perdeu seus ingressos deve iniciar o processo de compra desde o início.

