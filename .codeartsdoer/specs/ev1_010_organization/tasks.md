# EBC-X EV1-010 Organization 聚合根编码任务规划（tasks.md）

> **项目：EBC-X — Enterprise Business & Industrial Operating System（企业与工业智能运营操作系统）**
> **阶段：EV1 — Enterprise Core → EBCX-EV1-010 Organization 聚合根 → 编码任务规划（tasks.md）**
> **任务编号：EBCX-EV1-010**
> **文档版本：v1.0（首次生成，基于 spec.md v1.2 FINAL PASS / CLOSED + design.md v1.0 AUTHORIZED，转化为 16 个可执行、可验收、可追溯的施工任务 T01-T16）**
> **状态：🟡 TASK v1.0（待大G项目经理 EV1-010-TASK Gate Review）**
> **变更记录**：
> - v1.0（2026-09-09）：首次生成。基于 spec.md v1.2（FINAL PASS / CLOSED / 🔒 FROZEN）+ design.md v1.0（AUTHORIZED）转化为 16 个可执行、可验收、可追溯的施工任务（T01-T16），建立 RTM 矩阵，显式声明 4 项重点（pending 并发语义 / 2 条 Chain 首条记录 / 事务内固定锁顺序 / MoveOrganization 子树投影语义闭环），覆盖 R1.1-R1.4 + R2-R6 全部硬约束 + Design Mandatory #1/#2
> **需求基线（不可变）**：
> - `.codeartsdoer/specs/ev1_010_organization/spec.md` v1.2（EV1-010-SPEC FINAL PASS / CLOSED / 🔒 FROZEN，本任务规划不得修改，不得重新解释上位规范）
> - `.codeartsdoer/specs/ev1_010_organization/design.md` v1.0（EV1-010-DESIGN AUTHORIZED，本任务规划不得修改）
> - `.codeartsdoer/specs/ebcx_ev0_arch/spec.md` v1.1（EV0-SPEC PASS / CLOSED / 🔒 FROZEN）
> - `.codeartsdoer/specs/ebcx_ev0_arch/design.md` v1.1（EV0-DESIGN PASS / CLOSED / 🔒 FROZEN）
> - `.codeartsdoer/specs/ebcx_ev0_arch/tasks.md` v1.2（EV0-TASKS / 🔒 FROZEN）
> - `.codeartsdoer/specs/ev1_009_enterprise/spec.md` v1.0（EV1-009-SPEC PASS / CLOSED / 🔒 FROZEN）
> - `.codeartsdoer/specs/ev1_009_enterprise/design.md` v1.1（EV1-009-DESIGN PASS / CLOSED / 🔒 FROZEN）
> - `.codeartsdoer/specs/ev1_009_enterprise/tasks.md` v1.1（EV1-009-TASKS FINAL PASS / CLOSED / 🔒 FROZEN）
> **前置 Gate**：EV1-010-SPEC v1.2 = FINAL PASS / CLOSED，EV1-010-DESIGN v1.0 = AUTHORIZED，**当前授权：Task Definition**，**Coding = NOT AUTHORIZED**（需 Task Gate PASS 后才授权）
> **产品/架构总设计：大G项目经理体系**
> **核心工程化：华为云团队**
> **全球交付与产品主权：HTKIS**

---

## 文档定位与约束声明

本文档是 EV1-010 spec.md（v1.2，FINAL PASS / CLOSED / 🔒 FROZEN）+ design.md（v1.0，AUTHORIZED）的**施工任务化**展开，回答：

> **"怎样把已 PASS 的 design.md v1.0 转换为可执行、可验收、可追溯的编码任务，且不重新设计架构、不遗漏任一硬约束验收项、不侵入 EV1-009 / EV0 冻结边界。"**

**不可变基线约束**：
- ❌ 禁止重新设计架构（架构已 AUTHORIZED，本任务规划仅做"设计 → 施工任务"的 1:1 转换）
- ❌ 禁止修改 EV1-010 spec.md v1.2 / design.md v1.0（spec 已 FROZEN，design 已 AUTHORIZED）
- ❌ 禁止修改 EV0 spec.md / design.md / tasks.md（均已 FROZEN）
- ❌ 禁止修改 EV1-009 spec.md / design.md / tasks.md / `internal/enterprise/` 代码 / `business.enterprises` 表 / `EnterpriseCreated/Updated` 事件 / `EBCX-ENTERPRISE-*` 错误码（均已 FINAL CLOSED / FROZEN）
- ❌ 禁止扩大 EV1-010 Scope（仅 Organization 聚合根 + OrganizationTreeCoordinator 领域服务，不含 Person / MasterData / Permission / User / Authorization）
- ❌ 禁止在 Task 阶段进入 Coding（需 Task Gate PASS 后才授权）
- ❌ 禁止自行进入 EV1-011 或任何后续任务
- ✅ 仅定义 EV1-010 的施工任务、验收标准、依赖关系、RTM 追溯

**关联已 FROZEN 文档**：
- spec.md v1.2：§5.1 CreateOrganization / §5.2 UpdateOrganization / §5.3 MoveOrganization / §5.4 Evidence+Outbox 同事务 / §5.5 Neo4j 异步投影 / §6 数据约束 / §7 验收标准 / §8 硬约束 R1.1-R1.4+R2-R6 / §10.4 Q1-Q4-NORM / §11 禁止事项
- design.md v1.0：§二 增量设计方案 / §三 Design Mandatory 解决方案 / §四 HAS_CHILD Contract / §五 Code 唯一性 NULL 语义 / §六 EV1-009 兼容性 / §七 测试策略
- EV1-009 tasks.md v1.1：任务编号模式 T01-T13、任务粒度、测试分层、Evidence 要求（本 tasks.md 沿用其模式，扩展至 T01-T16）

---

## 大G项目经理 6 条硬约束继承声明（NON-NEGOTIABLE RED LINES）

本任务规划严格继承并落地 spec.md v1.2 §8 定义的 6 条硬约束（R1.1-R1.4 + R2-R6），任一遗漏将导致 EV1-010-TASK Gate 直接 REJECT：

| 约束编号 | 约束内容（spec.md v1.2 §8 原文摘要） | 落地任务 | 落地机制 | 验收 Test ID |
|---|---|---|---|---|
| **R1.1** | OrganizationAggregate 是 Business State（name、code）的唯一 Mutation Root | T02 / T08 / T09 / T11 | OrganizationAggregate 封装 CreateOrganization / UpdateOrganization 业务属性 Mutation 入口，外部仅通过 CommandHandler → Aggregate 路径，禁止绕过聚合根直接写 business.organizations 的 name/code 字段 | T13-U01 / T13-U02 / T14-I01 |
| **R1.2** | OrganizationTreeCoordinator 是 Tree Structural State（level、parentId）的 Mutation Authority | T03 / T10 / T11 | OrganizationTreeCoordinator.MoveSubtree() 在单一 `*sql.Tx` 内：(1) CAS UPDATE 被移动 Organization 的 parent_id/level/version；(2) 直接 SQL UPDATE 所有 descendant 的 level；(3) 同事务写入 2 条 Evidence + 1 条 Outbox Event。Coordinator 不写任何 Organization 的 name/code | T13-U08 / T13-U13 / T14-I06 / T14-I14 |
| **R1.3** | descendant level 变更不经过 Aggregate Mutation 方法，不递增 descendant version | T03 / T10 / T14 | descendant level 变更路径：Coordinator.MoveSubtree() → `UPDATE business.organizations SET level = ... WHERE org_id IN (descendant_ids)`，不调用 descendant OrganizationAggregate 的任何 Mutation 方法，不递增 descendant version | T13-U12 / T14-I14 |
| **R1.4** | MoveOrganization 产生 2 条 Evidence（Organization MOVE + OrganizationTree SUBTREE_STRUCTURAL_UPDATE） | T06 / T07 / T10 | Evidence Adapter 在同事务内写入 2 条 evidence.Record：第 1 条 chain_id="organization-mutation-chain-{tenantId}"，第 2 条 chain_id="organization-tree-structural-chain-{tenantId}"，两条 CorrelationID 相同 | T14-I13 / T15-P01 |
| **R2** | Idempotency + Organization + Evidence + Outbox 必须同一 ACID Transaction（四者原子） | T01 / T05 / T08 / T09 / T10 | UnitOfWork 在单一 `*sql.Tx` 内顺序执行：Idempotency Record 预留 → 聚合根/子树状态写入 → Evidence Ledger INSERT（1 或 2 条）→ Outbox Event INSERT → Idempotency Record MarkSuccess → COMMIT，任一失败整体 ROLLBACK | T14-I01 / T14-I09 |
| **R3** | Neo4j 永远不得进入 Organization Mutation 主事务 | T05 / T08 / T09 / T10 / T12 / T15 | 主事务（`*sql.Tx`）内仅访问 PostgreSQL；Neo4j 投影由独立 ProjectionConsumer 异步消费 Outbox Event，主事务 COMMIT 后才触发，Neo4j 故障不阻塞主事务 | T15-P05 |
| **R4** | Update / Move 必须实现 version-based CAS 乐观并发控制 | T04 / T09 / T10 | UpdateOrganization / MoveOrganization 使用原子 SQL `UPDATE ... SET version = version + 1 WHERE org_id = ? AND version = ?`，通过 `affectedRows == 0` 判定 CONCURRENCY_CONFLICT，禁止 SELECT-then-UPDATE 非原子方案 | T14-I05 |
| **R5** | 组织树层级 ≤5 不变式 | T02 / T03 / T08 / T10 | 聚合根在 CreateOrganization 时校验 parent.level + 1 ≤ 5，MoveOrganization 时校验子树最大深度 + 新 level ≤ 5，违反则拒绝 | T13-U03 / T13-U10 / T14-I07 |
| **R6** | Organization 必须归属于已存在的 Enterprise，enterpriseId 创建后不可变更 | T02 / T04 / T08 / T09 / T10 | CreateOrganization 时校验 enterpriseId 在 business.enterprises 表中存在（只读引用，受 RLS 隔离），UpdateOrganization / MoveOrganization 禁止变更 enterpriseId | T13-U02 / T13-U04 / T13-U11 |

---

## 重点 ①：Command Idempotency pending 并发语义声明（R2 落地，Coding 阶段硬性约束）

> **声明目的**：当第一个请求正在执行（status=pending）、第二个相同 commandId 同时到达时，数据库锁等待/唯一键冲突后的行为**在此写死**，禁止到 Coding 阶段临时决定。复用 EV1-009 tasks.md 重点 ① 的全部语义，仅替换错误码前缀。

### ①.1 UNIQUE(tenant_id, command_id) 冲突时的错误码与调用方行为

**场景**：两个并发请求携带相同 commandId 同时到达（同租户），均未命中已有 success/failed 记录。

**数据库行为**：
- 第一个请求执行 `INSERT INTO business.command_idempotency (command_id, tenant_id, ..., status='pending')` 成功获取 UNIQUE 行
- 第二个请求执行相同 INSERT 时违反 `UNIQUE(tenant_id, command_id)` 约束，PostgreSQL 抛出 `SQLSTATE 23505`（unique_violation）

**应用层行为（写死）**：
- `CommandIdempotencyRepository.CheckAndReserve` 捕获 `SQLSTATE 23505` → 返回 `ErrIdempotencyConflict`
- Handler 收到 `ErrIdempotencyConflict` → 不重试、不等待 → 直接向调用方返回错误码 **`EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT`**（HTTP 409 Conflict）
- 调用方行为契约：收到 409 Conflict 时，**应当重新查询当前命令状态**，而非盲目重试 POST
- **禁止服务端自动轮询等待 pending 完成**（避免 goroutine 泄漏与超时堆积）

### ①.2 pending 状态的超时处理

**处理策略（写死，复用 EV1-009 模式）**：
- **不引入后台清理 goroutine**（归属 EV1-022 运维任务）
- **依赖 PostgreSQL 事务超时机制**：pending 行所在事务若未 COMMIT，PostgreSQL 连接断开时自动 ROLLBACK
- **应用层显式超时**：Handler 通过 `context.WithTimeout(ctx, 30s)` 控制命令处理总时长（MoveOrganization 可配置 60s，因含子树递归 CTE + 批量 UPDATE）
- **调用方重试契约**：调用方收到超时错误后，复用同一 commandId 重试

### ①.3 并发同 commandId 的确定性响应矩阵

| 第一个请求状态 | 第二个请求行为 | 第二个请求响应 |
|---|---|---|
| 不存在（未到达） | INSERT 成功，继续执行 Mutation | 正常返回聚合根状态 |
| status=pending（执行中） | INSERT 违反 UNIQUE → `ErrIdempotencyConflict` | HTTP 409 `EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT` |
| status=success（已完成） | 命中首次结果，短路返回 | HTTP 200 + 首次聚合根状态 |
| status=failed（已失败） | DELETE 旧记录 + INSERT 新 pending，继续执行 | 正常返回新聚合根状态 或 新错误码 |

---

## 重点 ②：2 条 Evidence Chain 首条记录声明（R1.4 / REQUIRED-02 落地，Coding 阶段硬性约束）

> **声明目的**：明确 2 条独立 chain（organization-mutation-chain + organization-tree-structural-chain）的首条记录构造规则。**复用 EV1-003 的 `GenesisHash(chainID)`，禁止 EV1-010 自己创造第二套 Hash Chain 规则。复用 EV1-009 tasks.md 重点 ② 的 Advisory Lock 机制。**

### ②.0 Chain Advisory Lock（空链并发序列化，写死）

**Advisory Lock 获取规则（写死，复用 EV1-009 模式）**：
- **获取时机**：在同事务内、`SELECT ... FOR UPDATE`（Chain Predecessor lock）**之前**执行
- **获取语句**：`SELECT pg_advisory_xact_lock(hashtext($chainID))`
- **事务级**：COMMIT / ROLLBACK 时自动释放，无需手动 unlock，无泄漏风险
- **锁粒度**：chain 级别（按 chain_id hash），不同 chain 互不阻塞
- **2 条 chain 的 lock 顺序**：MoveOrganization 时先获取 chain 1（organization-mutation-chain）的 advisory lock，再获取 chain 2（organization-tree-structural-chain）的 advisory lock，防止死锁

### ②.1 chainID 粒度

- **chain 1**：`organization-mutation-chain-{tenantId}`（Organization 聚合根 Mutation Evidence，Create/Update/Move 第 1 条）
- **chain 2**：`organization-tree-structural-chain-{tenantId}`（子树结构变更 Structural Mutation Evidence，Move 第 2 条）
- **建链时机**：首次 Organization Mutation 时隐式建链，无显式 `CREATE CHAIN` DDL
- **链生命周期**：与租户生命周期一致，无显式销毁

### ②.2 首条 Evidence Record 的构造规则（写死）

**判定条件**：在同事务内先执行 `SELECT pg_advisory_xact_lock(hashtext($chainID))` 获取 chain 级别事务锁，再执行 `SELECT sequence_no, evidence_hash FROM evidence.evidence_ledger WHERE chain_id = $1 ORDER BY sequence_no DESC LIMIT 1 FOR UPDATE`，返回 0 行即为首条。

| 字段 | 首条取值 | 说明 |
|---|---|---|
| SequenceNo | **1** | 首条序号固定为 1 |
| PreviousEvidenceHash | **`GenesisHash(chainID)`** | 复用 EV1-003 `evidence.GenesisHash(chainID)` 函数 |
| EvidenceHash | `ChainHash(chainID, 1, GenesisHash(chainID), payload, sourceEventID, transactionID)` | SHA-256 |

### ②.3 非首条 Evidence Record 的构造规则（写死）

| 字段 | 非首条取值 | 说明 |
|---|---|---|
| SequenceNo | **N + 1** | 上一条 SequenceNo + 1 |
| PreviousEvidenceHash | **H[N]** | 上一条的 EvidenceHash |
| EvidenceHash | `ChainHash(chainID, N+1, H[N], payload, sourceEventID, transactionID)` | SHA-256 |

### ②.4 2 条 chain 独立性声明

- 2 条 chain 各自独立维护 Hash Chain（PreviousEvidenceHash 连续），**不跨 chain 链接**
- 2 条 chain 的 SequenceNo 各自单调递增，不共享序号空间
- `VerifyChain(records)` 按 chain_id 分组分别校验，不跨 chain 校验
- MoveOrganization 的 2 条 Evidence 在同一 ACID 事务内原子写入，任一失败整体 ROLLBACK（R2）

### ②.5 禁止事项

- ❌ 禁止 EV1-010 自定义 Genesis Hash 规则（必须复用 EV1-003 `GenesisHash(chainID)`）
- ❌ 禁止首条 SequenceNo = 0（必须为 1）
- ❌ 禁止跳过 `SELECT pg_advisory_xact_lock(hashtext($chainID))` 直接 `SELECT ... FOR UPDATE`
- ❌ 禁止跳过 `SELECT ... FOR UPDATE` 行锁直接 INSERT
- ❌ 禁止跨 chain 复用 PreviousEvidenceHash（每条 chain 独立）
- ❌ 禁止使用 session 级 `pg_advisory_lock`（必须用事务级 `pg_advisory_xact_lock`）
- ❌ 禁止 2 条 chain 的 advisory lock 获取顺序不固定（必须先 chain 1 后 chain 2，防止死锁）

---

## 重点 ③：事务内固定锁顺序声明（R2 落地，Coding 阶段硬性约束）

> **声明目的**：保持固定锁顺序，防止死锁。Create/Update 沿用 EV1-009 锁顺序，Move 新增子树批量 UPDATE 与 2 条 chain lock。

### ③.1 CreateOrganization 锁顺序（写死，禁止漂移）

```
BEGIN
 ↓
[SQL 0] Idempotency Record INSERT（business.command_idempotency，UNIQUE 冲突判定）
 ↓
[SQL 1] business.enterprises 只读查询（CheckEnterpriseExists，R6，无 FOR UPDATE）
 ↓
[SQL 1.1] business.organizations parent 只读查询（FindByID(parentId)，level 计算，无 FOR UPDATE）
 ↓
[SQL 1.5] Chain Advisory Lock（pg_advisory_xact_lock(hashtext("organization-mutation-chain-{tenantId}"))）
 ↓
[SQL 2] Evidence Chain predecessor lock（SELECT ... FOR UPDATE）
 ↓
[SQL 3] Evidence Record INSERT（evidence.evidence_ledger，1 条）
 ↓
[SQL 3.5] business.organizations INSERT（新行写入，无行锁冲突）
 ↓
[SQL 4] Outbox Event INSERT（outbox.events）
 ↓
[SQL 5] Idempotency Result UPDATE（status='success'）
 ↓
COMMIT
```

### ③.2 UpdateOrganization 锁顺序（写死，禁止漂移）

```
BEGIN
 ↓
[SQL 0] Idempotency Record INSERT
 ↓
[SQL 1] business.organizations 行锁（SELECT ... WHERE org_id = ? FOR UPDATE，加载现有聚合根）
 ↓
[SQL 1.5] Chain Advisory Lock（organization-mutation-chain）
 ↓
[SQL 2] Evidence Chain predecessor lock（SELECT ... FOR UPDATE）
 ↓
[SQL 3] Evidence Record INSERT（1 条）
 ↓
[SQL 3.5] business.organizations CAS UPDATE（UPDATE ... WHERE org_id = ? AND version = ?，行锁 + CAS）
 ↓
[SQL 4] Outbox Event INSERT
 ↓
[SQL 5] Idempotency Result UPDATE
 ↓
COMMIT
```

### ③.3 MoveOrganization 锁顺序（写死，禁止漂移，含子树批量 UPDATE + 2 条 chain lock）

```
BEGIN
 ↓
[SQL 0] Idempotency Record INSERT
 ↓
[SQL 1] business.organizations 被移动 Org 行锁（SELECT ... WHERE org_id = ? FOR UPDATE）
 ↓
[SQL 1.1] business.organizations newParent 只读查询（FindByID(newParentId)，无 FOR UPDATE）
 ↓
[SQL 1.2] business.organizations 子树查询（GetSubtree(orgId) 递归 CTE，无 FOR UPDATE）
 ↓
[SQL 2] business.organizations CAS UPDATE 被移动 Org（SET parent_id, level, version=version+1 WHERE org_id=? AND version=?，行锁 + CAS）
 ↓
[SQL 2.5] business.organizations 批量 UPDATE descendant level（SET level = level + levelDelta WHERE org_id IN (descendantIds)，批量行锁，不递增 version R1.3）
 ↓
[SQL 3] Chain 1 Advisory Lock（pg_advisory_xact_lock(hashtext("organization-mutation-chain-{tenantId}"))）
 ↓
[SQL 3.1] Chain 1 predecessor lock（SELECT ... FOR UPDATE）
 ↓
[SQL 3.2] Evidence Record 1 INSERT（organization-mutation-chain，第 1 条 Evidence）
 ↓
[SQL 4] Chain 2 Advisory Lock（pg_advisory_xact_lock(hashtext("organization-tree-structural-chain-{tenantId}"))）
 ↓
[SQL 4.1] Chain 2 predecessor lock（SELECT ... FOR UPDATE）
 ↓
[SQL 4.2] Evidence Record 2 INSERT（organization-tree-structural-chain，第 2 条 Evidence）
 ↓
[SQL 5] Outbox Event INSERT（OrganizationMoved）
 ↓
[SQL 6] Idempotency Result UPDATE
 ↓
COMMIT
```

### ③.4 死锁预防说明

- 全部事务按上述固定顺序获取锁，无锁顺序倒置，无死锁可能
- MoveOrganization 的 2 条 chain advisory lock 顺序固定（先 chain 1 后 chain 2），不同 chain_id 的 advisory lock 不冲突
- 批量 UPDATE descendant 在 CAS UPDATE 被移动 Org 之后、Evidence 写入之前，descendant 行未被其他步骤锁定，无死锁
- 并发 Move 涉及重叠子树时，步骤 1 的 FOR UPDATE 行锁会序列化，无死锁
- **本声明为 Coding 阶段硬性约束**，禁止在 Coding 阶段调整锁顺序

---

## 重点 ④：MoveOrganization 子树投影语义闭环声明（Design Mandatory #2 落地，Coding 阶段硬性约束）

> **声明目的**：明确 MoveOrganization 子树投影的 6 个子问题的 Coding 阶段实现约束，对齐 design.md v1.0 §三、3.2 完整闭环方案。

### ④.1 OrganizationMoved 事件不含 affectedDescendantIds[]（写死）

- OrganizationMoved 事件 payload 仅含被移动 Org 信息（orgId / oldParentId / newParentId / oldLevel / newLevel / version）
- **禁止**在事件 payload 中包含 affectedDescendantIds[] 列表（design.md D-002 决策）
- Projection Consumer 消费事件时通过递归 CTE 查询 PostgreSQL 重建子树

### ④.2 第 2 条 Evidence payload 含 affectedDescendantIds[]（写死）

- 第 2 条 Evidence（aggregateType="OrganizationTree", mutationType="SUBTREE_STRUCTURAL_UPDATE"）的 payload 含：
  - `subtreeRootOrgId`：被移动 Organization 的 orgId
  - `affectedDescendantIds[]`：子树所有 descendant 的 orgId 列表
  - `oldLevelOffset` / `newLevelOffset` / `levelDelta`
- 这是 Evidence（事实记录），不是领域事件（驱动投影）

### ④.3 Projection Consumer 使用绝对值 SET（写死）

- Projection Consumer 查询 PostgreSQL 获取子树所有 descendant 的当前正确 level
- Cypher 使用 `SET o.level = dl.level`（绝对值 SET），**禁止**使用 `SET o.level = o.level + $delta`（相对值 SET，重复 Replay 累加错误）
- 使用 `UNWIND $descendantLevels AS dl` 批量参数化更新

### ④.4 HAS_CHILD 边软关闭而非删除（写死）

- Move 时旧 HAS_CHILD 边 `SET r.validity = 'inactive'`（软关闭，保留历史）
- **禁止** `DELETE r`（直接删除，丢失历史）
- 新 HAS_CHILD 边 `MERGE (newParent)-[r:HAS_CHILD]->(moved)` 创建（validity='active'）
- 子树内部 HAS_CHILD 边不变（子树内部关系不变）

### ④.5 Reconciliation 以 PostgreSQL 为最终裁决源（写死）

- PostgreSQL → Neo4j 不一致时，以 PostgreSQL Evidence Ledger 为准
- Neo4j 通过 Event Replay / Reconciliation 重建
- Reconciliation 检测指标：missing_nodes / extra_nodes / property_mismatch / subtree_level_mismatch / has_child_edge_mismatch

---

# 一、施工任务清单（T01-T16）

> **任务编号规范**：`EV1-010-T01` ~ `EV1-010-T16`，主任务 16 个，无更深层级嵌套（Maximum 2 Levels）。
> **任务依赖图**：T01 → T02 → T03 / T04 / T05 / T06 / T07 → T08 / T09 / T10 → T11 → T12 → T13 / T14 → T15 → T16。
> **验收标准格式**：每个任务包含可测试的验收条件（checkbox），对应 Test ID 与 Evidence Item。

## 1. 数据库基础设施层

### EV1-010-T01：数据库迁移（business.organizations + UNIQUE NULLS NOT DISTINCT + 索引 + RLS + command_idempotency CHECK 扩展）

**任务描述**：创建 EV1-010 所需的全部 PostgreSQL 表结构、唯一索引（UNIQUE NULLS NOT DISTINCT）、行级安全策略，并兼容性扩展 command_idempotency 表的 CHECK 约束，为聚合根持久化与命令幂等提供数据库基础。

**输入**：
- design.md v1.0 §2.4.1 business.organizations 表结构
- design.md v1.0 §2.4.2 UNIQUE NULLS NOT DISTINCT 方案
- design.md v1.0 §2.4.3 RLS 策略
- design.md v1.0 §2.4.4 索引设计
- design.md v1.0 §2.4.5 command_idempotency CHECK 扩展
- spec.md v1.2 §6.1 数据约束
- EV1-009 V10 迁移（business.enterprises + business.command_idempotency 已创建）

**输出**：
- `db/migrations/V11__organization_aggregate.sql`

**验收标准**：
- [ ] `business.organizations` 表创建，字段含 org_id (UUID PK DEFAULT gen_random_uuid()) / enterprise_id (UUID NOT NULL) / parent_id (UUID NULL) / name (VARCHAR 256 NOT NULL) / code (VARCHAR 64 NOT NULL) / level (INTEGER NOT NULL) / version (BIGINT NOT NULL DEFAULT 1) / source_evidence_id (UUID NULL) / tenant_id (UUID NOT NULL) / created_at (TIMESTAMPTZ NOT NULL DEFAULT now()) / updated_at (TIMESTAMPTZ NOT NULL DEFAULT now())
- [ ] CHECK 约束 `chk_org_level CHECK (level BETWEEN 1 AND 5)` 创建（R5 层级约束）
- [ ] CHECK 约束 `chk_org_version CHECK (version >= 1)` 创建
- [ ] CHECK 约束 `chk_org_name CHECK (length(btrim(name)) > 0 AND length(name) <= 256)` 创建
- [ ] CHECK 约束 `chk_org_code CHECK (length(btrim(code)) > 0 AND length(code) <= 64)` 创建
- [ ] 外键 `fk_org_parent FOREIGN KEY (parent_id) REFERENCES business.organizations(org_id)` 创建（parent_id 自引用）
- [ ] 外键 `fk_org_enterprise FOREIGN KEY (enterprise_id) REFERENCES business.enterprises(enterprise_id)` 创建（R6 Enterprise 归属）
- [ ] UNIQUE 索引 `idx_organizations_enterprise_parent_code ON business.organizations (enterprise_id, parent_id, code) NULLS NOT DISTINCT` 创建（Code 唯一性 NULL 语义方案 A，裁决 #3）
- [ ] 索引 `idx_organizations_enterprise_parent ON business.organizations (enterprise_id, parent_id)` 创建
- [ ] 索引 `idx_organizations_level ON business.organizations (level)` 创建
- [ ] 索引 `idx_organizations_tenant ON business.organizations (tenant_id)` 创建
- [ ] 索引 `idx_organizations_enterprise ON business.organizations (enterprise_id)` 创建
- [ ] RLS 策略 `organizations_tenant_isolation ON business.organizations USING (tenant_id::text = current_setting('app.tenant_id', true)) WITH CHECK (...)` 创建
- [ ] Runtime Role 对 `business.organizations` 授予 SELECT / INSERT / UPDATE 权限，无 DELETE（组织为长生命周期对象）
- [ ] `business.command_idempotency` 表 CHECK 约束 `chk_idem_cmd_type` 扩展为 `CHECK (command_type IN ('CreateEnterprise', 'UpdateEnterprise', 'CreateOrganization', 'UpdateOrganization', 'MoveOrganization'))`（兼容性扩展，不破坏既有 Enterprise 幂等记录）
- [ ] `evidence.evidence_ledger` 表结构不修改（aggregate_type 为 VARCHAR 无 CHECK 约束，应用层使用新取值 "Organization" / "OrganizationTree"）
- [ ] 迁移脚本幂等（可重复执行不报错，使用 IF NOT EXISTS / DROP IF EXISTS）
- [ ] 迁移脚本在真实 PostgreSQL 18.3 上执行成功
- [ ] **禁止修改 V10__enterprise_aggregate.sql**（EV1-009 FROZEN）

**依赖任务**：无（基础设施层，最先执行）

**对应 Design 章节**：§2.4.1 表结构 / §2.4.2 UNIQUE NULLS NOT DISTINCT / §2.4.3 RLS / §2.4.4 索引 / §2.4.5 command_idempotency 扩展

**对应硬约束**：R2（同事务原子写入基础表） / R5（层级约束） / R6（Enterprise 归属外键） / 裁决 #3（UNIQUE NULLS NOT DISTINCT）

**对应 Test ID**：T14-I01（同事务原子） / T14-I02（DB UNIQUE 约束） / T15-P12（UNIQUE NULLS NOT DISTINCT Physical 验证）

---

## 2. 领域模型层

### EV1-010-T02：OrganizationAggregate 领域模型（Business State Mutation Root，R1.1）

**任务描述**：实现 OrganizationAggregate 聚合根、命令值对象（Create/Update）、领域事件（Created/Updated）、错误码枚举，封装全部 Business State 不变式校验与命令处理逻辑（非贫血模型，R1.1 落地）。**OrganizationAggregate 不含 Move 方法**（Move 由 OrganizationTreeCoordinator 处理，R1.2，Design Mandatory #1）。

**输入**：
- design.md v1.0 §2.2.2.1 OrganizationAggregate 接口签名
- design.md v1.0 §2.5.1 OrganizationAggregate 设计
- design.md v1.0 §2.5.3 Command 定义（CreateOrganizationCommand / UpdateOrganizationCommand）
- design.md v1.0 §2.5.4 Event 定义（OrganizationCreatedEvent / OrganizationUpdatedEvent）
- design.md v1.0 §2.5.5 不变式校验汇总
- spec.md v1.2 §5.1 / §5.2 业务规则
- spec.md v1.2 §6.1 / §6.2 / §6.3 数据约束

**输出**：
- `internal/organization/aggregate.go`（OrganizationAggregate 结构体 + CreateOrganization / UpdateOrganization 方法）
- `internal/organization/command.go`（CreateOrganizationCommand / UpdateOrganizationCommand / MoveOrganizationCommand 值对象）
- `internal/organization/event.go`（OrganizationCreatedEvent / OrganizationUpdatedEvent / OrganizationMovedEvent）
- `internal/organization/errors.go`（OrganizationError 错误码枚举）

**验收标准**：
- [ ] `OrganizationAggregate` 结构体含 OrgID / EnterpriseID / ParentID / Name / Code / Level / Version / SourceEvidenceID / TenantID / CreatedAt / UpdatedAt 字段
- [ ] `CreateOrganizationCommand` 含 CommandID / EnterpriseID / ParentID / Name / Code / TenantID / SourceEvidenceID 字段
- [ ] `UpdateOrganizationCommand` 含 CommandID / OrgID / NewName / NewCode / ExpectedVersion / SourceEvidenceID 字段
- [ ] `MoveOrganizationCommand` 含 CommandID / OrgID / NewParentID / ExpectedVersion / SourceEvidenceID 字段
- [ ] `OrganizationCreatedEvent` 含 EventID / EventType="OrganizationCreated" / OrgID / EnterpriseID / ParentID / Name / Code / Level / Version / SourceEvidenceID / TenantID / Timestamp / TraceID 字段
- [ ] `OrganizationUpdatedEvent` 含 EventID / EventType="OrganizationUpdated" / OrgID / EnterpriseID / ParentID / NewName / NewCode / Level / Version / SourceEvidenceID / TenantID / Timestamp / TraceID 字段
- [ ] `OrganizationMovedEvent` 含 EventID / EventType="OrganizationMoved" / OrgID / EnterpriseID / OldParentID / NewParentID / OldLevel / NewLevel / Version / SourceEvidenceID / TenantID / Timestamp / TraceID 字段
- [ ] `OrganizationMovedEvent` **不含** affectedDescendantIds[] 列表（Design Mandatory #2 问题 1 决策，重点 ④.1 写死）
- [ ] `OrganizationAggregate.CreateOrganization(cmd)` 执行不变式校验（name 非空 1~256、code 非空 1~64、commandId 非空 UUID v4），校验通过产出 OrganizationCreatedEvent（version=1），校验失败返回对应错误码
- [ ] `OrganizationAggregate.UpdateOrganization(cmd)` 执行版本单调递增校验（ExpectedVersion > 0），校验通过产出 OrganizationUpdatedEvent（version=ExpectedVersion+1），校验失败返回 EBCX-ORGANIZATION-VERSION-MONOTONIC-VIOLATION
- [ ] `OrganizationAggregate` **不含 Move() 方法**（Design Mandatory #1，Move 由 OrganizationTreeCoordinator 处理 R1.2）
- [ ] `OrganizationAggregate.UpdateOrganization` 禁止变更 parentId / level / enterpriseId（仅更新 name / code，R1.1）
- [ ] level 在 CreateOrganization 时由 Aggregate 根据 parentId 计算（level = parent.level + 1 或 level = 1），禁止命令参数直接设置 level
- [ ] 错误码枚举含 EBCX-ORGANIZATION-NOT-FOUND / EBCX-ORGANIZATION-VERSION-CONFLICT / EBCX-ORGANIZATION-VERSION-MONOTONIC-VIOLATION / EBCX-ORGANIZATION-LEVEL-EXCEED-MAX / EBCX-ORGANIZATION-SUBTREE-LEVEL-EXCEED-MAX / EBCX-ORGANIZATION-CYCLE-DETECTED / EBCX-ORGANIZATION-PARENT-NOT-FOUND / EBCX-ORGANIZATION-PARENT-CROSS-ENTERPRISE / EBCX-ORGANIZATION-ENTERPRISE-NOT-FOUND / EBCX-ORGANIZATION-ENTERPRISE-IMMUTABLE / EBCX-ORGANIZATION-DUPLICATE-CODE / EBCX-ORGANIZATION-INVALID-NAME / EBCX-ORGANIZATION-INVALID-CODE / EBCX-ORGANIZATION-INVALID-COMMAND-ID / EBCX-ORGANIZATION-COMMAND-ID-REQUIRED / EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT / EBCX-ORGANIZATION-UNKNOWN-COMMAND / EBCX-ORGANIZATION-LEVEL-FORBIDDEN-SET
- [ ] 聚合根为非贫血模型（封装不变式校验与命令处理逻辑，非仅 getter/setter）
- [ ] 领域层无外部依赖（纯 Go，不依赖 database/sql / evidence / outbox，依赖方向单向向内）
- [ ] **禁止修改 `internal/enterprise/` 任何代码**（EV1-009 FROZEN）

**依赖任务**：EV1-010-T01（需表结构存在以理解字段约束）

**对应 Design 章节**：§2.2.2.1 OrganizationAggregate / §2.5.1 设计 / §2.5.3 Command / §2.5.4 Event / §2.5.5 不变式

**对应硬约束**：R1.1（Business State Mutation Root） / R5（层级 ≤5） / R6（Enterprise 归属）

**对应 Test ID**：T13-U01 / T13-U02 / T13-U03 / T13-U04 / T13-U05 / T13-U06 / T13-U07

---

### EV1-010-T03：OrganizationTreeCoordinator 领域服务（Tree Structural State Mutation Authority，R1.2）

**任务描述**：实现 OrganizationTreeCoordinator 领域服务（Domain Service，非 Aggregate Root），含 MoveSubtree() 跨聚合树结构协调操作方法，在单一 ACID 事务内原子更新整个子树的 level + 被移动 Organization 的 parentId/version。落地 Design Mandatory #1（MoveOrganization 流程统一为 CommandHandler → Coordinator.MoveSubtree()）。

**输入**：
- design.md v1.0 §2.2.2.2 OrganizationTreeCoordinator 接口签名
- design.md v1.0 §2.5.2 OrganizationTreeCoordinator 设计
- design.md v1.0 §2.1.3.1 MoveOrganization 流程图
- design.md v1.0 §三、3.1 Design Mandatory #1 解决方案
- spec.md v1.2 §5.3 业务规则 / §10.4 Q1-Q4-NORM
- EV1-010-T02（OrganizationAggregate + Command + Event）

**输出**：
- `internal/organization/tree_coordinator.go`（OrganizationTreeCoordinator 结构体 + MoveSubtree 方法）

**验收标准**：
- [ ] `OrganizationTreeCoordinator` 结构体含 repo / evWriter / structEvWriter / outbox 依赖字段
- [ ] `NewOrganizationTreeCoordinator(repo, evWriter, structEvWriter, outbox)` 构造函数，依赖注入
- [ ] `MoveSubtree(ctx, tx, cmd)` 方法签名对齐 design.md §2.2.2.2，返回 `(*OrganizationMovedEvent, *SubtreeUpdateResult, error)`
- [ ] MoveSubtree 内部执行步骤对齐 design.md §2.5.2（10 步：FindByID → CAS 校验 → FindByID(newParent) + 同企业校验 + 无环校验 → GetSubtree → 子树层级校验 → CAS UPDATE 被移动 Org → 批量 UPDATE descendant level → WriteMutationEvidence → WriteStructuralEvidence → Outbox.Write）
- [ ] 无环校验算法：从 newParentId 向上遍历 parentId 链至根，若途中遇到 orgId 则拒绝（ErrOrgCycleDetected），复杂度 O(n) n≤5
- [ ] 子树层级校验算法：subtreeMaxDepth = max(depth)，若 subtreeMaxDepth + newLevel > 5 则拒绝（ErrOrgSubtreeLevelExceedMax）
- [ ] **Coordinator 不写任何 Organization 的 name / code**（R1.2，业务属性归 OrganizationAggregate R1.1）
- [ ] **Coordinator 不调用 descendant OrganizationAggregate 的任何 Mutation 方法**（R1.3，descendant level 变更通过直接 SQL UPDATE）
- [ ] **descendant level 变更不递增 descendant version**（R1.3 / Q3-NORM，通过 `UPDATE business.organizations SET level = level + levelDelta WHERE org_id IN (descendantIds)`，不递增 version）
- [ ] 被移动 Organization 的 version 递增（通过 CAS UPDATE `SET version = version + 1 WHERE org_id = ? AND version = ?`，R4）
- [ ] 2 条 Evidence 在同一 ACID 事务内原子写入（Q4-NORM / R1.4），分别属于 2 条独立 chain
- [ ] Coordinator 无持久化状态（Domain Service，无状态）
- [ ] **禁止 OrganizationAggregate.Move() 方法定义**（Design Mandatory #1，编译期 lint 拒绝）

**依赖任务**：EV1-010-T02（OrganizationAggregate + Command + Event）

**对应 Design 章节**：§2.2.2.2 OrganizationTreeCoordinator / §2.5.2 设计 / §2.1.3.1 流程图 / §三、3.1 Design Mandatory #1

**对应硬约束**：R1.2（Tree Structural State Mutation Authority） / R1.3（descendant version 不递增） / R1.4（2 条 Evidence） / R4（CAS） / R5（子树层级 ≤5）

**对应 Test ID**：T13-U08 / T13-U09 / T13-U10 / T13-U11 / T13-U12 / T13-U13

---

## 3. 持久化与事务边界层

### EV1-010-T04：OrganizationRepository 接口与 PostgreSQL 实现（含 CAS + GetSubtree + UpdateSubtreeLevels + CheckEnterpriseExists，R4 落地）

**任务描述**：实现聚合根持久化抽象接口与 PostgreSQL 实现，含 7 个方法：Insert / FindByID / UpdateWithCAS / ExistsByCode / GetSubtree / UpdateSubtreeLevels / CheckEnterpriseExists。GetSubtree 使用递归 CTE，UpdateSubtreeLevels 使用批量 UPDATE，UpdateWithCAS 使用原子 SQL 实现 R4 乐观并发控制。

**输入**：
- design.md v1.0 §2.2.2.3 OrganizationRepository 接口签名
- design.md v1.0 §2.6.2 OrganizationPostgreSQLRepository 实现
- design.md v1.0 §2.6.1 OrganizationRepository 接口
- spec.md v1.2 §6.1 数据约束
- EV1-010-T01（business.organizations 表已创建）

**输出**：
- `internal/organization/repository/organization.go`（OrganizationRepository 接口）
- `internal/organization/repository/organization_postgresql.go`（OrganizationRepositoryPostgreSQL 实现）

**验收标准**：
- [ ] `OrganizationRepository` 接口含 7 个方法：`Insert(ctx, tx, agg)` / `FindByID(ctx, tx, orgID, forUpdate)` / `UpdateWithCAS(ctx, tx, orgID, newName, newCode, expectedVersion, sourceEvidenceID)` / `ExistsByCode(ctx, tx, enterpriseID, parentID, code, excludeOrgID)` / `GetSubtree(ctx, tx, orgID)` / `UpdateSubtreeLevels(ctx, tx, descendantIDs, levelDelta)` / `CheckEnterpriseExists(ctx, tx, enterpriseID)`
- [ ] `Insert` 执行 `INSERT INTO business.organizations (...) VALUES (...)`，违反 UNIQUE NULLS NOT DISTINCT → 映射为 ErrOrgDuplicateCode
- [ ] `FindByID` 执行 `SELECT ... FROM business.organizations WHERE org_id = $1 [FOR UPDATE]`，支持 forUpdate 参数控制行锁
- [ ] `UpdateWithCAS` 执行原子 SQL `UPDATE business.organizations SET name=$1, code=$2, version=version+1, source_evidence_id=$3, updated_at=now() WHERE org_id=$4 AND version=$5 RETURNING ...`（R4 原子 CAS）
- [ ] `UpdateWithCAS` 检查 affectedRows：==1 → CAS 成功；==0 → 诊断（行存在且 version≠expectedVersion → ErrOrgVersionConflict，行不存在 → ErrOrgNotFound）
- [ ] `ExistsByCode` 执行 `SELECT EXISTS(SELECT 1 FROM business.organizations WHERE enterprise_id=$1 AND parent_id IS NOT DISTINCT FROM $2 AND code=$3 AND ($4::uuid IS NULL OR org_id <> $4::uuid))`（parent_id IS NOT DISTINCT FROM 确保 NULL 语义正确）
- [ ] `GetSubtree` 执行递归 CTE 查询子树所有后代（含 depth），返回 `[]SubtreeNode`，每个 SubtreeNode 含 OrgID / Level / Depth
- [ ] `UpdateSubtreeLevels` 执行批量 `UPDATE business.organizations SET level = level + $1, updated_at = now() WHERE org_id = ANY($2::uuid[])`（**不递增 version**，R1.3 / Q3-NORM）
- [ ] `CheckEnterpriseExists` 执行 `SELECT EXISTS(SELECT 1 FROM business.enterprises WHERE enterprise_id = $1)`（只读引用 business.enterprises，R6，受 RLS 隔离）
- [ ] 所有方法接受外部 `*sql.Tx` 参数，不自开事务（R2 同事务原子）
- [ ] **禁止 SELECT-then-UPDATE 非原子方案**（R4 硬性约束）
- [ ] **禁止修改 `internal/enterprise/` 任何代码**（EV1-009 FROZEN）

**依赖任务**：EV1-010-T01（表结构） / EV1-010-T02（聚合根与错误码）

**对应 Design 章节**：§2.2.2.3 OrganizationRepository / §2.6.1 接口 / §2.6.2 实现

**对应硬约束**：R4（原子 CAS） / R2（同事务原子） / R6（Enterprise 归属只读引用） / R1.3（UpdateSubtreeLevels 不递增 version）

**对应 Test ID**：T14-I01（同事务原子） / T14-I05（CAS 并发冲突） / T14-I06（子树递归更新） / T14-I02（DB UNIQUE 约束）

---

### EV1-010-T05：UnitOfWork 事务边界（复用 EV1-009 模式，R2 落地）

**任务描述**：复用 EV1-009 的 UnitOfWork 接口与实现，不新建。Organization Handler 通过依赖注入接收 `repository.UnitOfWork` 接口（EV1-009 定义），运行时注入 `UnitOfWorkPostgreSQL` 实例。

**输入**：
- design.md v1.0 §2.6.3 UnitOfWork 扩展
- design.md v1.0 §六 兼容性声明（复用 EV1-009 UnitOfWork）
- EV1-009 `internal/enterprise/repository/unit_of_work.go`（已 CLOSED，不修改源码）
- EV1-008 RLSManager（已 CLOSED）

**输出**：
- 无新建文件（复用 EV1-009 UnitOfWork 接口与实现）
- `internal/organization/handler/` 的 Handler 通过依赖注入接收 `repository.UnitOfWork` 接口

**验收标准**：
- [ ] Organization Handler 通过依赖注入接收 `repository.UnitOfWork` 接口（EV1-009 定义）
- [ ] 运行时注入 `UnitOfWorkPostgreSQL` 实例（EV1-009 实现）
- [ ] `BeginTenantTx` 内部调用 `RLSManager.BeginTenantTransaction(ctx, rlsCtx)`，返回绑定 RLS 上下文的 `*sql.Tx`
- [ ] 单一 `*sql.Tx` 贯穿 Idempotency + Organization + Evidence + Outbox 四次写入（R2 四者原子，Move 含 2 条 Evidence 为五次写入）
- [ ] Neo4j 不在 `*sql.Tx` 内（R3，事务内无 Neo4j driver 调用）
- [ ] 禁止嵌套事务或独立连接
- [ ] Handler 编排顺序遵循本 tasks.md 重点 ③ 固定锁顺序
- [ ] 应用层显式超时：Create/Update Handler `context.WithTimeout(ctx, 30s)`，Move Handler `context.WithTimeout(ctx, 60s)`（含子树递归 CTE + 批量 UPDATE）
- [ ] **禁止修改 EV1-009 `internal/enterprise/repository/unit_of_work.go` 源码**（EV1-009 FROZEN）

**依赖任务**：EV1-010-T01（表结构） / EV1-008（RLSManager） / EV1-009（UnitOfWork 接口）

**对应 Design 章节**：§2.6.3 UnitOfWork 扩展 / §六 兼容性声明

**对应硬约束**：R2（四者同事务原子） / R3（Neo4j 不在主事务）

**对应 Test ID**：T14-I01（同事务原子） / T14-I09（Idempotency 同事务原子） / T15-P05（Neo4j 故障隔离）

---

## 4. Evidence 写入适配层

### EV1-010-T06：OrganizationEvidenceWriter 适配器（chain 1: organization-mutation-chain，R1.4 落地）

**任务描述**：实现 Organization Mutation 到 evidence.Record 的适配写入（第 1 条 Evidence，chain_id = "organization-mutation-chain-{tenantId}"），复用 EV1-003 ChainHash 计算 hash，在同事务内 INSERT 至 evidence.evidence_ledger。复用 EV1-009 Evidence Writer 模式（advisory lock + SELECT FOR UPDATE + ChainHash）。

**输入**：
- design.md v1.0 §2.8.1 复用 EV1-009 Evidence Writer 模式
- design.md v1.0 §2.8.2 扩展 aggregate_type 枚举
- design.md v1.0 §2.8.3 MoveOrganization 的 2 条 Evidence 写入方案（第 1 条）
- design.md v1.0 §2.8.4 Chain Hash 维护
- 本 tasks.md 重点 ②（2 条 Chain 首条记录声明）
- EV1-003 evidence.Record / ChainHash / GenesisHash / VerifyChain / DetectTamper（已 CLOSED）
- EV1-009 `internal/enterprise/evidence_adapter/writer_impl.go`（参考模式，不修改源码）

**输出**：
- `internal/organization/evidence_adapter/writer.go`（OrganizationEvidenceWriter 接口）
- `internal/organization/evidence_adapter/writer_impl.go`（OrganizationEvidenceWriter 实现）

**验收标准**：
- [ ] `OrganizationEvidenceWriter` 接口含 `Write(ctx, tx, agg, event, mutationType)` 方法，返回 `*evidence.Record`
- [ ] chain_id = `"organization-mutation-chain-{tenantId}"`（第 1 条 Evidence chain）
- [ ] aggregateType = `"Organization"`（Create/Update/Move 的第 1 条 Evidence）
- [ ] mutationType 取值 "CREATE" / "UPDATE" / "MOVE"
- [ ] Evidence Record 构造遵循 design.md §2.8 字段取值（EvidenceID / ChainID / SequenceNo / PreviousEvidenceHash / EvidenceHash / EvidenceType / Payload / SourceEventID / TransactionID / TenantID / Provenance / CorrelationID / CausationID / CreatedBy / CreatedAt / Version / LegalHold）
- [ ] **Chain Advisory Lock 获取**（重点 ②.0 写死）：先执行 `SELECT pg_advisory_xact_lock(hashtext($chainID))`，再执行 `SELECT ... FOR UPDATE`
- [ ] **Advisory Lock 事务级释放**：使用 `pg_advisory_xact_lock`（事务级），禁止 `pg_advisory_lock`（session 级）
- [ ] 首条 Evidence Record：SequenceNo=1，PreviousEvidenceHash=`GenesisHash(chainID)`（复用 EV1-003，重点 ②.2 写死）
- [ ] 非首条 Evidence Record：SequenceNo=N+1，PreviousEvidenceHash=H[N]（重点 ②.3 写死）
- [ ] EvidenceHash 由 `evidence.ChainHash(chainID, sequenceNo, previousHash, payload, sourceEventID, transactionID)` 计算（复用 EV1-003）
- [ ] Provenance 字段含 git_commit / trace_id / event_id / evidence_id / operator，git_commit 与实际验证代码 commit 一致
- [ ] 方法接受外部 `*sql.Tx` 参数，不自开事务（R2 同事务原子）
- [ ] Runtime Role 仅 INSERT + SELECT，无 UPDATE/DELETE（D-GATE-01 append-only 四层防御）
- [ ] **禁止 EV1-010 自定义 Genesis Hash 规则**（必须复用 EV1-003 `GenesisHash(chainID)`，重点 ②.5 写死）
- [ ] **禁止修改 EV1-009 `internal/enterprise/evidence_adapter/` 任何代码**（EV1-009 FROZEN）

**依赖任务**：EV1-010-T02（聚合根与事件） / EV1-003（evidence 基础设施）

**对应 Design 章节**：§2.8.1 复用 EV1-009 模式 / §2.8.2 扩展 aggregate_type / §2.8.3 第 1 条 Evidence / §2.8.4 Chain Hash

**对应硬约束**：R2（同事务原子） / R1.4（Evidence 产生规则） / TASK-R04（Evidence First）

**对应 Test ID**：T14-I03（Evidence Ledger 记录字段完整） / T14-I10（Hash Chain 完整性） / T15-P01（Hash Chain Physical 验证）

---

### EV1-010-T07：OrganizationTreeStructuralEvidenceWriter 适配器（chain 2: organization-tree-structural-chain，R1.4 落地）

**任务描述**：实现子树结构变更到 evidence.Record 的适配写入（第 2 条 Evidence，chain_id = "organization-tree-structural-chain-{tenantId}"），承载 MoveOrganization 的子树结构变更事实记录。payload 含 affectedDescendantIds[] + oldLevelOffset + newLevelOffset + levelDelta。

**输入**：
- design.md v1.0 §2.8.3 MoveOrganization 的 2 条 Evidence 写入方案（第 2 条）
- design.md v1.0 §2.8.4 Chain Hash 维护
- design.md v1.0 §三、3.2.2 问题 2（第 2 条 Evidence payload 结构）
- 本 tasks.md 重点 ②（2 条 Chain 首条记录声明）
- EV1-010-T06（OrganizationEvidenceWriter 模式参考）
- EV1-003 evidence.Record / ChainHash / GenesisHash（已 CLOSED）

**输出**：
- `internal/organization/evidence_adapter/structural_writer.go`（OrganizationTreeStructuralEvidenceWriter 接口）
- `internal/organization/evidence_adapter/structural_writer_impl.go`（OrganizationTreeStructuralEvidenceWriter 实现）

**验收标准**：
- [ ] `OrganizationTreeStructuralEvidenceWriter` 接口含 `Write(ctx, tx, subtreeRootOrgID, affectedDescendantIDs, oldLevelOffset, newLevelOffset, event, sourceEvidenceID)` 方法，返回 `*evidence.Record`
- [ ] chain_id = `"organization-tree-structural-chain-{tenantId}"`（第 2 条 Evidence chain，独立于 chain 1）
- [ ] aggregateType = `"OrganizationTree"`（子树结构变更 Evidence）
- [ ] mutationType = `"SUBTREE_STRUCTURAL_UPDATE"`
- [ ] payload 含 `subtreeRootOrgId` / `affectedDescendantIds[]` / `oldLevelOffset` / `newLevelOffset` / `levelDelta` / `tenantId` / `sourceEvidenceId` / `movedOrgOldParentId` / `movedOrgNewParentId` / `movedOrgOldLevel` / `movedOrgNewLevel` / `movedOrgNewVersion`（design.md §三、3.2.2 字段说明）
- [ ] `affectedDescendantIds[]` 含子树所有 descendant 的 orgId 列表（不含被移动 Organization 自身），由 GetSubtree() 返回
- [ ] `levelDelta = newLevelOffset - oldLevelOffset`
- [ ] **Chain Advisory Lock 获取**（重点 ②.0 写死）：先执行 `SELECT pg_advisory_xact_lock(hashtext($chainID))`，再执行 `SELECT ... FOR UPDATE`
- [ ] 首条 Evidence Record：SequenceNo=1，PreviousEvidenceHash=`GenesisHash(chainID)`（复用 EV1-003）
- [ ] 非首条 Evidence Record：SequenceNo=N+1，PreviousEvidenceHash=H[N]
- [ ] EvidenceHash 由 `evidence.ChainHash()` 计算（复用 EV1-003）
- [ ] **2 条 Evidence 的 CorrelationID 相同**（同一次 MoveOrganization 操作的 UUID）
- [ ] **2 条 Evidence 的 CausationID 指向同一条 OrganizationMoved 领域事件**（eventID）
- [ ] **2 条 chain 独立维护 Hash Chain**（不跨 chain 链接，重点 ②.4 写死）
- [ ] 方法接受外部 `*sql.Tx` 参数，不自开事务（R2 同事务原子）
- [ ] Runtime Role 仅 INSERT + SELECT，无 UPDATE/DELETE
- [ ] **禁止跨 chain 复用 PreviousEvidenceHash**（重点 ②.5 写死）

**依赖任务**：EV1-010-T02（聚合根与事件） / EV1-010-T06（Writer 模式参考） / EV1-003（evidence 基础设施）

**对应 Design 章节**：§2.8.3 第 2 条 Evidence / §2.8.4 Chain Hash / §三、3.2.2 问题 2

**对应硬约束**：R1.4（2 条 Evidence） / R2（同事务原子） / TASK-R04（Evidence First）

**对应 Test ID**：T14-I13（2 条 Evidence 同事务原子） / T14-I10（Hash Chain 完整性） / T15-P01（Hash Chain Physical 验证）

---

## 5. 应用服务编排层

### EV1-010-T08：CreateOrganizationHandler 应用服务编排（R1.1 / R2 / R6 落地）

**任务描述**：实现 CreateOrganizationHandler 应用层命令处理器，编排 RLS 事务开启 → 幂等预留 → 聚合根 CreateOrganization → Enterprise 存在性校验 → parentId 层级校验 → code 唯一性校验 → Repository Insert → Evidence 写入（1 条）→ Outbox 写入 → 幂等 MarkSuccess → 事务提交。

**输入**：
- design.md v1.0 §2.7.1 CreateOrganizationHandler 处理流程
- design.md v1.0 §2.1.3.2 CreateOrganization 流程
- 本 tasks.md 重点 ① / ② / ③
- EV1-010-T02 / T04 / T05 / T06

**输出**：
- `internal/organization/handler/create_handler.go`（CreateOrganizationHandler）

**验收标准**：
- [ ] `CreateOrganizationHandler` 依赖 UnitOfWork / OrganizationRepository / OrganizationEvidenceWriter / outbox.Publisher / CommandIdempotencyRepository
- [ ] `CreateOrganizationHandler.Handle(ctx, cmd)` 编排顺序：BeginTenantTx → idemRepo.CheckAndReserve → agg.CreateOrganization(cmd) → repo.CheckEnterpriseExists（R6）→ repo.FindByID(parentId)（若非空，加载 parent 计算 level）→ repo.ExistsByCode → repo.Insert → evWriter.Write（1 条 Evidence）→ outbox.Write → idemRepo.MarkSuccess → uow.Commit
- [ ] Idempotency 命中 success 短路返回首次结果，命中 failed DELETE 重试，UNIQUE 冲突返回 EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT（重点 ①.1 写死）
- [ ] 任一环节失败 → uow.Rollback(tx)，已写入的 SQL 自动撤销（含 Idempotency Record，R2）
- [ ] Outbox Event payload 含 evidenceRef 字段（= 同事务写入的 Evidence Record.evidence_id，1:1 对应）
- [ ] Outbox Event payload 含 orgId / enterpriseId / parentId / name / code / level / version / sourceEvidenceId / tenantId / timestamp / traceId 字段
- [ ] Handler 不含业务逻辑（业务逻辑在聚合根，Handler 仅负责编排与基础设施调用）
- [ ] 应用层显式超时：`context.WithTimeout(ctx, 30s)`（重点 ①.2）
- [ ] tenantId 由 RLS 上下文注入，禁止命令参数覆盖（cmd.TenantID 必须等于 RLS 上下文 tenantId，否则返回 EBCX-TENANT-CONTEXT-MISMATCH）
- [ ] level 由聚合根根据 parentId 计算（parentId 非空时 level = parent.level + 1，空时 level = 1），禁止命令参数直接设置 level
- [ ] **禁止修改 `internal/enterprise/` 任何代码**（EV1-009 FROZEN）

**依赖任务**：EV1-010-T02 / T04 / T05 / T06

**对应 Design 章节**：§2.7.1 CreateOrganizationHandler / §2.1.3.2 流程

**对应硬约束**：R1.1（经 Aggregate） / R2（四者同事务原子） / R6（Enterprise 归属校验） / R5（层级校验）

**对应 Test ID**：T14-I01（同事务原子） / T14-I03（Evidence 记录字段完整） / T14-I04（Outbox 异步发布） / T14-I11（Idempotency 幂等命中）

---

### EV1-010-T09：UpdateOrganizationHandler 应用服务编排（R1.1 / R2 / R4 落地）

**任务描述**：实现 UpdateOrganizationHandler 应用层命令处理器，编排加载 → 聚合根 UpdateOrganization → code 唯一性校验（排除自身）→ Repository UpdateWithCAS → Evidence 写入（1 条）→ Outbox 写入 → 幂等 MarkSuccess → 事务提交。

**输入**：
- design.md v1.0 §2.7.2 UpdateOrganizationHandler 处理流程
- design.md v1.0 §2.1.3.2 UpdateOrganization 流程
- 本 tasks.md 重点 ① / ② / ③
- EV1-010-T02 / T04 / T05 / T06

**输出**：
- `internal/organization/handler/update_handler.go`（UpdateOrganizationHandler）

**验收标准**：
- [ ] `UpdateOrganizationHandler` 依赖 UnitOfWork / OrganizationRepository / OrganizationEvidenceWriter / outbox.Publisher / CommandIdempotencyRepository
- [ ] `UpdateOrganizationHandler.Handle(ctx, cmd)` 编排顺序：BeginTenantTx → idemRepo.CheckAndReserve → repo.FindByID(orgId) → agg.UpdateOrganization(cmd) → repo.ExistsByCode（排除自身）→ repo.UpdateWithCAS（R4 原子 CAS）→ evWriter.Write（1 条 Evidence）→ outbox.Write → idemRepo.MarkSuccess → uow.Commit
- [ ] **UpdateOrganization 仅更新 name / code**（Business State，R1.1），不变更 parentId / level / enterpriseId
- [ ] CAS 并发冲突（affectedRows == 0）→ 返回 EBCX-ORGANIZATION-VERSION-CONFLICT（R4）
- [ ] 任一环节失败 → uow.Rollback(tx)（R2）
- [ ] Outbox Event payload 含 evidenceRef 字段
- [ ] 应用层显式超时：`context.WithTimeout(ctx, 30s)`
- [ ] **禁止变更 parentId / level / enterpriseId**（R1.1，parentId/level 归 Coordinator R1.2，enterpriseId 不可变更 R6）

**依赖任务**：EV1-010-T02 / T04 / T05 / T06

**对应 Design 章节**：§2.7.2 UpdateOrganizationHandler / §2.1.3.2 流程

**对应硬约束**：R1.1（经 Aggregate） / R2（四者同事务原子） / R4（CAS）

**对应 Test ID**：T14-I01（同事务原子） / T14-I05（CAS 并发冲突） / T14-I11（Idempotency 幂等命中）

---

### EV1-010-T10：MoveOrganizationHandler 应用服务编排（R1.2 / R2 / R4 / R5 落地，Design Mandatory #1 核心）

**任务描述**：实现 MoveOrganizationHandler 应用层命令处理器，编排 RLS 事务开启 → 幂等预留 → **OrganizationTreeCoordinator.MoveSubtree()**（含加载被移动 Organization → CAS 校验 → newParent 校验 → 无环校验 → 子树层级校验 → CAS UPDATE 被移动 Organization → 批量 UPDATE descendant level → Evidence 写入 2 条 → Outbox 写入）→ 幂等 MarkSuccess → 事务提交。**Design Mandatory #1 核心：Handler 不调用 OrganizationAggregate 的任何方法，而是调用 Coordinator.MoveSubtree()**。

**输入**：
- design.md v1.0 §2.7.3 MoveOrganizationHandler 处理流程
- design.md v1.0 §2.1.3.1 MoveOrganization 流程图
- design.md v1.0 §三、3.1 Design Mandatory #1 解决方案
- 本 tasks.md 重点 ① / ② / ③ / ④
- EV1-010-T03 / T04 / T05 / T06 / T07

**输出**：
- `internal/organization/handler/move_handler.go`（MoveOrganizationHandler）

**验收标准**：
- [ ] `MoveOrganizationHandler` 依赖 UnitOfWork / OrganizationTreeCoordinator / CommandIdempotencyRepository
- [ ] `MoveOrganizationHandler.Handle(ctx, cmd)` 编排顺序：BeginTenantTx → idemRepo.CheckAndReserve → **coordinator.MoveSubtree(ctx, tx, cmd)**（Coordinator 内部完成全部 Move 逻辑）→ idemRepo.MarkSuccess → uow.Commit
- [ ] **Handler 不调用 OrganizationAggregate 的任何方法**（Design Mandatory #1，重点 ④ 落地）
- [ ] **Handler 不含 Move 业务逻辑**（Move 逻辑全在 Coordinator.MoveSubtree() 内，Handler 仅负责事务边界 + 幂等 + 编排）
- [ ] Coordinator.MoveSubtree() 内部完成：FindByID → CAS 校验 → FindByID(newParent) + 同企业校验 + 无环校验 → GetSubtree → 子树层级校验 → CAS UPDATE 被移动 Org → 批量 UPDATE descendant level → WriteMutationEvidence（第 1 条）→ WriteStructuralEvidence（第 2 条）→ Outbox.Write
- [ ] **2 条 Evidence 在同一 ACID 事务内原子写入**（Q4-NORM / R1.4），CorrelationID 相同
- [ ] **descendant level 变更不递增 descendant version**（R1.3 / Q3-NORM）
- [ ] CAS 并发冲突（affectedRows == 0）→ 返回 EBCX-ORGANIZATION-VERSION-CONFLICT（R4）
- [ ] 无环校验失败 → 返回 EBCX-ORGANIZATION-CYCLE-DETECTED
- [ ] 子树层级超限 → 返回 EBCX-ORGANIZATION-SUBTREE-LEVEL-EXCEED-MAX
- [ ] 跨企业引用 → 返回 EBCX-ORGANIZATION-PARENT-CROSS-ENTERPRISE
- [ ] 任一环节失败 → uow.Rollback(tx)（R2）
- [ ] Outbox Event payload 含 evidenceRef 字段（指向第 1 条 Evidence 的 evidence_id）
- [ ] OrganizationMoved 事件 **不含** affectedDescendantIds[]（Design Mandatory #2 问题 1，重点 ④.1 写死）
- [ ] 应用层显式超时：`context.WithTimeout(ctx, 60s)`（含子树递归 CTE + 批量 UPDATE）
- [ ] **禁止变更 enterpriseId**（R6，MoveOrganization 命令不含 enterpriseId 参数）

**依赖任务**：EV1-010-T03 / T04 / T05 / T06 / T07

**对应 Design 章节**：§2.7.3 MoveOrganizationHandler / §2.1.3.1 流程图 / §三、3.1 Design Mandatory #1

**对应硬约束**：R1.2（经 Coordinator） / R1.3（descendant version 不递增） / R1.4（2 条 Evidence） / R2（四者同事务原子） / R4（CAS） / R5（子树层级 ≤5） / R6（enterpriseId 不可变更）

**对应 Test ID**：T14-I06（子树递归更新） / T14-I07（子树层级校验） / T14-I08（子树 level 递归更新同事务原子） / T14-I13（2 条 Evidence 同事务原子） / T14-I14（descendant version 不递增）

---

### EV1-010-T11：OrganizationCommandBus 命令分发

**任务描述**：实现命令分发器，路由 CreateOrganization / UpdateOrganization / MoveOrganization 命令至对应 Handler，作为聚合根命令的统一入口。

**输入**：
- design.md v1.0 §2.1.2 组件架构（OrganizationCommandBus）
- design.md v1.0 §2.2.2 接口清单
- EV1-010-T08 / T09 / T10（Handler 已实现）

**输出**：
- `internal/organization/handler/command_bus.go`（OrganizationCommandBus）

**验收标准**：
- [ ] `OrganizationCommandBus` 含 `Dispatch(ctx, cmd)` 方法，接受 CreateOrganizationCommand / UpdateOrganizationCommand / MoveOrganizationCommand
- [ ] `Dispatch` 根据命令类型路由至 CreateOrganizationHandler.Handle / UpdateOrganizationHandler.Handle / MoveOrganizationHandler.Handle
- [ ] 未知命令类型 → 返回 EBCX-ORGANIZATION-UNKNOWN-COMMAND
- [ ] CommandBus 不含业务逻辑（仅路由）
- [ ] CommandBus 是聚合根命令的统一入口（外部仅通过 CommandBus → Handler → Aggregate/Coordinator 路径，R1.1/R1.2 落地）

**依赖任务**：EV1-010-T08 / T09 / T10

**对应 Design 章节**：§2.1.2 组件架构 / §2.2.2 接口清单

**对应硬约束**：R1.1 / R1.2（唯一 Mutation 入口）

**对应 Test ID**：T13-U01 / T13-U08（通过 CommandBus 触发命令）

---

## 6. Graph 投影规则扩展层

### EV1-010-T12：Organization 投影规则注册 + HAS_CHILD 边 + 子树投影 + Reconciliation（R3 / Design Mandatory #2 落地）

**任务描述**：向 EV1-005 ProjectionRuleRegistry 追加 `organization.created` / `organization.updated` / `organization.moved` 三条投影规则，MERGE Organization 节点 + BELONGS_TO 边 + HAS_CHILD 边。实现 MoveOrganization 子树投影完整方案（Design Mandatory #2）：Projection Consumer 查询 PostgreSQL 重建子树 + Cypher SET 批量更新（绝对值 SET）+ HAS_CHILD 边软关闭/创建 + Reconciliation 机制。扩展 EdgeHasChild 枚举值。

**输入**：
- design.md v1.0 §2.9 Graph Projection 设计
- design.md v1.0 §2.9.1 Organization 节点 Contract
- design.md v1.0 §2.9.2 BELONGS_TO 边 Contract
- design.md v1.0 §2.9.3 HAS_CHILD 边 Contract
- design.md v1.0 §2.9.4 MoveOrganization 子树投影完整方案
- design.md v1.0 §2.9.5 Event Replay 方案
- design.md v1.0 §2.9.6 Reconciliation 机制
- design.md v1.0 §三、3.2 Design Mandatory #2 完整闭环方案
- design.md v1.0 §四 HAS_CHILD 边 Contract 正式定义
- 本 tasks.md 重点 ④（MoveOrganization 子树投影语义闭环声明）
- EV1-005 ProjectionRuleRegistry（已 CLOSED，仅追加规则项）
- EV1-014 24 Entity Graph Schema（已 CLOSED，NodeOrganization 节点契约）

**输出**：
- `internal/organization/projection/register_rules.go`（向 ProjectionRuleRegistry 追加 organization.* 规则的初始化函数）
- `internal/organization/projection/has_child_edge.go`（HAS_CHILD 边投影逻辑）
- `internal/organization/projection/subtree_projection.go`（子树 level 批量投影逻辑）
- `internal/organization/projection/reconciliation.go`（Reconciliation 机制）

**验收标准**：
- [ ] 向 `ProjectionRuleRegistry.rules` 追加 `organization.created` 规则：MERGE Organization 节点 + MERGE BELONGS_TO 边（Organization→Enterprise）+ MERGE HAS_CHILD 边（parent→child，若 parentId 非空）
- [ ] 向 `ProjectionRuleRegistry.rules` 追加 `organization.updated` 规则：MERGE Organization 节点属性更新（name、code、version）
- [ ] 向 `ProjectionRuleRegistry.rules` 追加 `organization.moved` 规则：SET Organization 节点 parentId/level/version + HAS_CHILD 边变更（旧边 validity='inactive'，新边创建）+ 子树 level 批量 SET
- [ ] 投影节点 nodeType='Organization'，符合 EV1-014 24 Entity Graph Schema 契约
- [ ] 投影节点 sourceEvidenceId 从 Outbox Event payload.evidenceRef 提取
- [ ] **HAS_CHILD 边 Contract 实现**（design.md §四）：
  - edgeType = 'HAS_CHILD'（扩展 Graph Edge，非 Canonical 8 类边之一）
  - source = Organization 节点（parent），target = Organization 节点（child）
  - provenance = { sourceEvent, sourceEvidenceId }
  - lifecycle = { validity: 'active' | 'inactive', createdAt, updatedAt, inactiveByEvent }
- [ ] **Move 时旧 HAS_CHILD 边软关闭**（重点 ④.4 写死）：`SET r.validity = 'inactive', r.updatedAt = $timestamp, r.inactiveByEvent = $eventId`，**禁止 DELETE r**
- [ ] **Move 时新 HAS_CHILD 边 MERGE 创建**：`MERGE (newParent)-[r:HAS_CHILD]->(moved) SET r.validity = 'active', ...`
- [ ] **子树内部 HAS_CHILD 边不变**（子树内部关系不变）
- [ ] **BELONGS_TO 边不变**（Organization 归属 Enterprise 不可变更，R6）
- [ ] **Projection Consumer 查询 PostgreSQL 重建子树**（Design Mandatory #2 问题 1，重点 ④.1 写死）：消费 OrganizationMoved 事件时，通过递归 CTE 查询 PostgreSQL 获取子树所有 descendant 的当前正确 level
- [ ] **Cypher SET 批量更新使用绝对值 SET**（重点 ④.3 写死）：`UNWIND $descendantLevels AS dl MATCH (o:Organization {orgId: dl.orgId}) SET o.level = dl.level`，**禁止相对值 SET**（`SET o.level = o.level + $delta`，重复 Replay 累加错误）
- [ ] **Event Replay 幂等**：MERGE 节点/边 + 绝对值 SET 属性，重复 Replay 结果一致
- [ ] **Reconciliation 机制实现**（design.md §2.9.6）：
  - 检测：比对 PostgreSQL 的 (level, parentId, version) 与 Neo4j 的 (level, parentId, version)
  - 修复：重放相关事件，ProjectionConsumer 使用 PostgreSQL 值覆盖 Neo4j
  - 指标：missing_nodes / extra_nodes / property_mismatch / subtree_level_mismatch / has_child_edge_mismatch
- [ ] **Reconciliation 以 PostgreSQL 为最终裁决源**（重点 ④.5 写死）
- [ ] 在 `internal/platform/graph/model.go` EdgeType 枚举中新增 `EdgeHasChild = "HAS_CHILD"`（扩展枚举值，不删除既有值）
- [ ] **不修改 EV1-005 基础设施源码契约**（仅追加规则项与枚举值）
- [ ] 投影异步执行，不阻塞主事务（R3）
- [ ] 投影消费者复用 EV1-005 IdempotentConsumer（基于 event_id 去重）

**依赖任务**：EV1-010-T08 / T09 / T10（Outbox Event 已产出） / EV1-005（ProjectionRuleRegistry） / EV1-014（NodeOrganization 契约）

**对应 Design 章节**：§2.9 Graph Projection / §三、3.2 Design Mandatory #2 / §四 HAS_CHILD Contract

**对应硬约束**：R3（异步投影，主事务外） / TASK-R06（边由 Domain Event 驱动） / D-GATE-06（24 Entity Graph Contract）

**对应 Test ID**：T15-P02（Neo4j Organization 节点投影） / T15-P03（BELONGS_TO 边投影） / T15-P04（HAS_CHILD 边投影） / T15-P05（投影最终一致 ≤3s） / T15-P08（子树投影一致性） / T15-P09（Event Replay 幂等） / T15-P10（Reconciliation） / T15-P11（HAS_CHILD 边一致性）

---

## 7. 测试层

### EV1-010-T13：Unit Test（聚合根不变式 + Coordinator 树操作 + commandId 幂等 + 事件产出 + 错误码）

**任务描述**：实现纯内存 Unit Test，覆盖 OrganizationAggregate 不变式校验、OrganizationTreeCoordinator 树操作逻辑、命令处理逻辑、事件产出、错误码映射、commandId 幂等命中短路返回，无外部依赖。

**输入**：
- design.md v1.0 §2.12.1 Unit Test 表
- design.md v1.0 §七、7.2 Unit Test 策略
- EV1-010-T02 / T03

**输出**：
- `internal/organization/aggregate_test.go`
- `internal/organization/tree_coordinator_test.go`

**验收标准**：
- [ ] `TestOrganizationAggregate_CreateOrganization_Invariants`（T13-U01）：name/code 非空/超长拒绝、version=1、产出 OrganizationCreatedEvent、commandId 非空 UUID v4 校验、level 计算正确
- [ ] `TestOrganizationAggregate_CreateOrganization_EnterpriseValidation`（T13-U02）：enterpriseId 不存在拒绝 ErrOrgEnterpriseNotFound
- [ ] `TestOrganizationAggregate_CreateOrganization_LevelValidation`（T13-U03）：parent.level=5 拒绝 ErrOrgLevelExceedMax
- [ ] `TestOrganizationAggregate_CreateOrganization_CrossEnterpriseParent`（T13-U04）：跨企业 parent 拒绝 ErrOrgParentCrossEnterprise
- [ ] `TestOrganizationAggregate_CreateOrganization_DuplicateCode`（T13-U05）：同企业同父 code 唯一 ErrOrgDuplicateCode
- [ ] `TestOrganizationAggregate_CreateOrganization_LevelCalculation`（T13-U06）：level = parent.level + 1 / level = 1
- [ ] `TestOrganizationAggregate_BoundaryLint`（T13-U07）：跨聚合直接调用 lint 拒绝
- [ ] `TestOrganizationAggregate_UpdateOrganization_Invariants`（T13-U08）：name/code 更新、version 递增
- [ ] `TestOrganizationAggregate_UpdateOrganization_VersionMonotonic`（T13-U09）：CAS 乐观锁 ErrOrgVersionConflict
- [ ] `TestOrganizationAggregate_UpdateOrganization_ForbiddenFields`（T13-U10）：禁止变更 parentId/level/enterpriseId
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_CommandProcessing`（T13-U11）：被移动 Org parentId/level/version 更新
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_CycleDetection`（T13-U12）：newParentId 是后代/自身拒绝 ErrOrgCycleDetected
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_SubtreeLevelValidation`（T13-U13）：subtreeMaxDepth + newLevel > 5 拒绝
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_CrossEnterprise`：跨企业 parent 拒绝
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_DescendantVersionNotIncrement`（T13-U14）：R1.3/Q3-NORM 验证 descendant version 不递增
- [ ] `TestOrganizationTreeCoordinator_MoveSubtree_CoordinatorNotWriteBusinessState`（T13-U15）：R1.2 验证 Coordinator 不写 name/code
- [ ] `TestOrganizationAggregate_NoMoveMethod`（T13-U16）：验证 OrganizationAggregate 不含 Move() 方法（Design Mandatory #1）
- [ ] 全部 Unit Test 在纯内存环境执行 PASS（无 PostgreSQL / Neo4j 依赖）
- [ ] 测试覆盖率 ≥ 90%（聚合根不变式 + Coordinator 树操作 + 命令处理 + 事件产出 + 错误码 + 幂等命中）

**依赖任务**：EV1-010-T02 / T03

**对应 Design 章节**：§2.12.1 Unit Test / §七、7.2 Unit Test 策略

**对应硬约束**：R1.1（聚合根不变式） / R1.2（Coordinator 树操作） / R1.3（descendant version 不递增） / R5（层级 ≤5） / R6（Enterprise 归属）

**对应 Test ID**：T13-U01 ~ T13-U16（共 17 个 Unit Test，对齐 design.md §2.12.1）

---

### EV1-010-T14：Integration Test（同事务原子 + Evidence Hash Chain + Outbox + RLS + CAS + 唯一性 + 子树递归更新 + 2 条 Evidence + Command Idempotency + 子树层级校验）

**任务描述**：实现 Integration Test，使用真实 PostgreSQL（可 Mock Neo4j），覆盖同事务原子写入、Evidence Hash Chain（2 条独立 chain）、Outbox 发布、RLS 租户隔离、CAS 并发冲突、组织唯一性（UNIQUE NULLS NOT DISTINCT）、MoveOrganization 子树递归更新、2 条 Evidence 同事务原子、Command Idempotency、子树层级校验、descendant version 不递增。

**输入**：
- design.md v1.0 §2.12.2 Integration Test 表
- design.md v1.0 §七、7.3 Integration Test 策略
- 本 tasks.md 重点 ① / ② / ③ / ④
- EV1-010-T01 ~ T12

**输出**：
- `internal/organization/handler/create_handler_integration_test.go`
- `internal/organization/handler/update_handler_integration_test.go`
- `internal/organization/handler/move_handler_integration_test.go`
- `internal/organization/evidence_adapter/writer_integration_test.go`
- `internal/organization/evidence_adapter/structural_writer_integration_test.go`
- `internal/organization/repository/organization_integration_test.go`

**验收标准**：
- [ ] `TestIntegration_SameTransactionAtomic`（T14-I01）：Organization + Evidence + Outbox 同事务 COMMIT，任一失败整体 ROLLBACK（R2 四者原子，含 Idempotency Record）
- [ ] `TestIntegration_UniqueCodeConstraint_NULLS_NOT_DISTINCT`（T14-I02）：UNIQUE NULLS NOT DISTINCT 生效，parentId IS NULL 时 code 唯一性约束生效
- [ ] `TestIntegration_EvidenceLedgerRecordFields`（T14-I03）：16 字段完整、aggregateType="Organization" / "OrganizationTree"
- [ ] `TestIntegration_OutboxWriteAndPublish`（T14-I04）：Outbox Event status pending → published，异步发布至 Kafka（可 Mock Kafka）
- [ ] `TestIntegration_CASConcurrencyConflict`（T14-I05）：并发 UpdateOrganization 基于同一 expectedVersion → 仅一个 affectedRows==1，其余 → EBCX-ORGANIZATION-VERSION-CONFLICT（R4）
- [ ] `TestIntegration_MoveSubtree_RecursiveUpdate`（T14-I06）：descendant level 递归更新、同事务原子
- [ ] `TestIntegration_MoveSubtree_SubtreeLevelValidation`（T14-I07）：subtreeMaxDepth + newLevel > 5 拒绝
- [ ] `TestIntegration_MoveSubtree_AtomicRollback`（T14-I08）：整体 ROLLBACK 无部分成功
- [ ] `TestIntegration_EvidenceOutbox_SameTransactionAtomic`（T14-I09）：ROLLBACK 无部分成功
- [ ] `TestIntegration_EvidenceHashChain_TwoIndependentChains`（T14-I10）：2 条 chain 各自 SequenceNo 连续、PreviousEvidenceHash 连续，不跨 chain 链接
- [ ] `TestIntegration_CommandIdempotency_DuplicateReturnsFirstResult`（T14-I11）：相同 commandId 重复提交 → 返回首次结果，不产生新 Evidence / Outbox Event
- [ ] `TestIntegration_RLSTenantIsolation`（T14-I12）：跨租户访问拒绝
- [ ] `TestIntegration_MoveSubtree_TwoEvidenceSameTransaction`（T14-I13）：2 条 Evidence CorrelationID 相同、任一失败整体 ROLLBACK（R1.4 / Q4-NORM）
- [ ] `TestIntegration_MoveSubtree_DescendantVersionNotIncrement`（T14-I14）：descendant version 不递增（R1.3 / Q3-NORM）
- [ ] 全部 Integration Test 在真实 PostgreSQL 18.3 上执行 PASS（Neo4j 可 Mock）
- [ ] 测试覆盖率 ≥ 85%

**依赖任务**：EV1-010-T01 ~ T12

**对应 Design 章节**：§2.12.2 Integration Test / §七、7.3 Integration Test 策略

**对应硬约束**：R1.3 / R1.4 / R2 / R4 / R5 / R6

**对应 Test ID**：T14-I01 ~ T14-I14（共 14 个 Integration Test，对齐 design.md §2.12.2）

---

### EV1-010-T15：Physical Test（真实 PG + Neo4j + 子树投影一致性 + Event Replay + Reconciliation + HAS_CHILD 边一致性 + Evidence JSON + Provenance）

**任务描述**：实现 Physical Test，使用真实 PostgreSQL 18.3 + Neo4j 5.x 验证，覆盖真实持久化、Graph 投影（Organization 节点 + BELONGS_TO + HAS_CHILD 边）、Neo4j 故障隔离、投影最终一致 ≤3s、**MoveOrganization 子树投影一致性**（Design Mandatory #2）、**Event Replay 幂等性**、**PostgreSQL→Neo4j Reconciliation**、**HAS_CHILD 边一致性**、**UNIQUE NULLS NOT DISTINCT Physical 验证**、Evidence Chain Hash Chain 完整性、Physical Evidence JSON 产出、Provenance 一致性。**禁止 Mock 冒充 Physical Evidence。**

**输入**：
- design.md v1.0 §2.12.3 Physical Test 表
- design.md v1.0 §七、7.4 Physical Test 策略
- design.md v1.0 §三、3.2 Design Mandatory #2 完整闭环方案
- spec.md v1.2 §4.4 可维护性 / §7.2 测试验收 / §7.3 Evidence 验收
- 本 tasks.md 重点 ① / ② / ④
- EV1-010-T01 ~ T14

**输出**：
- `internal/organization/physical_test.go`（或 `internal/organization/*_physical_test.go`，遵循 `TestPhysical_*` 命名规范）

**验收标准**：
- [ ] `TestPhysical_EvidenceHashChain_Integrity`（T15-P01）：从 Genesis 逐条验证 2 条 chain 各自 sequence 连续 / previousHash 连续 / hash 重算一致，复用 EV1-003 `VerifyChain` + `DetectTamper`
- [ ] `TestPhysical_Neo4j_OrganizationNodeProjection`（T15-P02）：真实 Neo4j 5.x 节点 MERGE 成功，含 NodeOrganization，nodeType='Organization'，属性符合 EV1-014 契约
- [ ] `TestPhysical_Neo4j_BELONGS_TOEdgeProjection`（T15-P03）：Organization→Enterprise 边、由 Domain Event 驱动（TASK-R06）
- [ ] `TestPhysical_Neo4j_HAS_CHILDEdgeProjection`（T15-P04）：parent→child 边、Move 时旧边 validity='inactive'/新边创建
- [ ] `TestPhysical_Neo4j_FaultIsolation_EventuallyConsistent`（T15-P05）：Neo4j 故障（停止容器）→ 主事务 COMMIT 成功，Outbox Event 待重投，主事务不受影响（R3）；恢复后投影最终一致 ≤3s
- [ ] `TestPhysical_EvidenceJSON_Produced`（T15-P06）：`evidence/ev1/EBCX-EV1-010-organization-evidence.json` 存在且字段完整（EV0 TASK-H07 14 个最小字段标准）
- [ ] `TestPhysical_ProvenanceConsistency`（T15-P07）：Evidence.git_commit = 实际验证代码 commit hash
- [ ] `TestPhysical_MoveSubtree_SubtreeProjectionConsistency`（T15-P08，Design Mandatory #2）：Neo4j descendant level 与 PostgreSQL 一致
- [ ] `TestPhysical_MoveSubtree_EventReplayIdempotent`（T15-P09，Design Mandatory #2）：重放 OrganizationMoved 事件，Neo4j 状态一致（MERGE + 绝对值 SET 幂等）
- [ ] `TestPhysical_Reconciliation_PGNeo4j`（T15-P10，Design Mandatory #2）：不一致检测 + 修复，PG 为最终裁决源
- [ ] `TestPhysical_MoveSubtree_HAS_CHILDEdgeConsistency`（T15-P11，Design Mandatory #2）：旧边 validity='inactive'、新边 validity='active'、子树内部边不变
- [ ] `TestPhysical_UniqueCode_NULLS_NOT_DISTINCT`（T15-P12，裁决 #3）：parentId IS NULL 时 code 唯一性生效
- [ ] 全部 Physical Test 在真实 PostgreSQL 18.3 + Neo4j 5.x 上执行 PASS
- [ ] **禁止 Mock 冒充 Physical Evidence**（spec §11 禁止事项 6，TASK-R09）
- [ ] **禁止 Test PASS 冒充 Physical PASS**（spec §11 禁止事项 7）
- [ ] 测试命名遵循 `TestPhysical_*` 格式

**依赖任务**：EV1-010-T01 ~ T14

**对应 Design 章节**：§2.12.3 Physical Test / §七、7.4 Physical Test 策略 / §三、3.2 Design Mandatory #2

**对应硬约束**：R3 / R1.4 / TASK-R09 / 裁决 #2 / 裁决 #3 / Design Mandatory #2

**对应 Test ID**：T15-P01 ~ T15-P12（共 12 个 Physical Test，对齐 design.md §2.12.3）

---

## 8. Physical Evidence 层

### EV1-010-T16：Physical Evidence JSON 产出

**任务描述**：产出 `evidence/ev1/EBCX-EV1-010-organization-evidence.json`，含聚合根测试、Evidence 记录、Graph 节点与边验证，符合 EV0 TASK-H07 14 个最小字段标准，作为大G项目经理 Gate Review 的物理证据。

**输入**：
- design.md v1.0 §七、7.4 Physical Evidence JSON 路径与 14 个最小字段标准
- spec.md v1.2 §1.3 核心输出 / §4.4 可维护性 / §7.3 Evidence 验收
- EV1-010-T13 / T14 / T15（测试结果汇总）

**输出**：
- `evidence/ev1/EBCX-EV1-010-organization-evidence.json`

**验收标准**：
- [ ] JSON 文件存在于 `evidence/ev1/EBCX-EV1-010-organization-evidence.json` 路径
- [ ] JSON 含 EV0 TASK-H07 14 个最小字段：execution_id / timestamp / environment / git_commit / test_command / actual_output / actual_metrics / database_state / event_id / evidence_id / trace_id / failure_injection_result / verification_result / verifier
- [ ] JSON 含 `taskId` 字段，值为 "EBCX-EV1-010"
- [ ] JSON 含 `taskName` 字段，值为 "Organization 聚合根"
- [ ] JSON 含 `gateName` 字段，值为 "EV1-010-CODING"
- [ ] JSON 含 `gitCommit` 字段（实际验证代码 commit hash，与 Evidence Record.git_commit 一致）
- [ ] JSON 含 `infrastructure` 字段，值为 "PostgreSQL 18.3 + Neo4j 5.x"（真实基础设施，非 Mock）
- [ ] JSON 含 `testLayer` 字段，值为 "unit" / "integration" / "physical"（三层测试结果汇总）
- [ ] JSON 含 `testNames` 字段（测试函数名列表）
- [ ] JSON 含 `result` 字段，值为 "PASS" / "FAIL"（汇总三层测试结果）
- [ ] JSON 含 `metrics` 字段（性能指标：Create/Update P95 ≤500ms、Move P95 ≤800ms、投影延迟 ≤3s、不变式校验 ≤10ms、子树校验 ≤50ms）
- [ ] JSON 含 `artifacts` 字段（证据附件路径：Evidence 记录、Graph 节点与边截图等）
- [ ] JSON 含 `rtm` 字段（Requirement Traceability Matrix，对应本 tasks.md RTM 章节）
- [ ] JSON 含 `lockOrder` 字段（事务内固定锁顺序声明，对应本 tasks.md 重点 ③）
- [ ] JSON 含 `pendingSemantics` 字段（commandId pending 并发语义声明，对应本 tasks.md 重点 ①）
- [ ] JSON 含 `chainGenesis` 字段（2 条 Chain 首条记录声明，对应本 tasks.md 重点 ②）
- [ ] JSON 含 `subtreeProjectionSemantics` 字段（MoveOrganization 子树投影语义闭环声明，对应本 tasks.md 重点 ④）
- [ ] JSON 含 `hardConstraints` 字段（R1.1-R1.4 + R2-R6 落地状态汇总）
- [ ] JSON 含 `designMandatory` 字段（Design Mandatory #1 / #2 落地状态汇总）
- [ ] **禁止 Mock 冒充 Physical Evidence**（infrastructure 字段必须为真实 PostgreSQL 18.3 + Neo4j 5.x）
- [ ] **禁止破坏 Provenance**（gitCommit 必须与实际验证代码 commit 一致，spec §11 禁止事项 12）

**依赖任务**：EV1-010-T13 / T14 / T15（三层测试全部 PASS 后产出）

**对应 Design 章节**：§七、7.4 Physical Evidence JSON / §2.12.3 Physical Test

**对应硬约束**：TASK-R09（Physical Claim 由真实 Evidence 支撑） / TASK-H07（Physical Evidence 最小字段标准）

**对应 Test ID**：T15-P06（Physical Evidence JSON 产出） / T15-P07（Provenance 一致性）

---

# 二、Requirement Traceability Matrix（RTM）

> **RTM 目的**：确保每个 Design Requirement → Task → Test ID → Evidence Item 完整可追溯，R1.1-R1.4 + R2-R6 每项都有对应的 Task + Test + Evidence。

## 2.1 硬约束 RTM

| 约束编号 | 约束内容 | Design 章节 | 落地 Task | 落地 Test ID | Evidence Item |
|---|---|---|---|---|---|
| R1.1 | OrganizationAggregate 是 Business State 唯一 Mutation Root | §2.2.2.1 / §2.5.1 | T02 / T08 / T09 / T11 | T13-U01 / T13-U02 / T13-U08 / T14-I01 | artifacts.aggregate_invariants / artifacts.mutation_root_only |
| R1.2 | OrganizationTreeCoordinator 是 Tree Structural State Mutation Authority | §2.2.2.2 / §2.5.2 / §三、3.1 | T03 / T10 / T11 | T13-U11 / T13-U15 / T14-I06 / T14-I14 | artifacts.coordinator_mutation_authority / artifacts.coordinator_not_write_business_state |
| R1.3 | descendant level 变更不经过 Aggregate Mutation 方法，不递增 descendant version | §2.5.2 / §2.7.3 / §2.10.3 | T03 / T10 / T14 | T13-U14 / T14-I14 | artifacts.descendant_version_not_increment / artifacts.passive_structural_state_change |
| R1.4 | MoveOrganization 产生 2 条 Evidence | §2.8.3 / §三、3.2.2 | T06 / T07 / T10 | T14-I13 / T15-P01 | artifacts.two_evidence_same_transaction / artifacts.two_independent_chains |
| R2 | Idempotency + Organization + Evidence + Outbox 同 ACID 事务 | §2.6.3 / §2.7.1-§2.7.3 | T01 / T05 / T08 / T09 / T10 | T14-I01 / T14-I09 | artifacts.same_transaction_atomic / artifacts.rollback_no_partial |
| R3 | Neo4j 不进入主事务 | §2.9.4 / §2.9.5 | T05 / T08 / T09 / T10 / T12 / T15 | T15-P05 | artifacts.neo4j_fault_isolation / artifacts.projection_eventually_consistent |
| R4 | Version CAS 乐观并发控制（原子 SQL） | §2.6.2 / §2.7.2 / §2.7.3 | T04 / T09 / T10 | T14-I05 | artifacts.cas_atomic_sql / artifacts.cas_conflict_detection |
| R5 | 组织树层级 ≤5 不变式 | §2.5.5 / §2.7.1 / §2.7.3 | T02 / T03 / T08 / T10 | T13-U03 / T13-U13 / T14-I07 | artifacts.level_le_5 / artifacts.subtree_level_validation |
| R6 | Organization 必须归属于已存在的 Enterprise，enterpriseId 创建后不可变更 | §2.5.5 / §2.7.1 | T02 / T04 / T08 / T09 / T10 | T13-U02 / T13-U04 / T13-U10 | artifacts.enterprise_ownership / artifacts.enterprise_id_immutable |

## 2.2 红线 RTM

| 红线编号 | 红线内容 | Design 章节 | 落地 Task | 落地 Test ID | Evidence Item |
|---|---|---|---|---|---|
| TASK-R03 | 聚合根边界（禁止跨聚合直接调用） | §2.1.2 | T02 / T03 / T11 | T13-U07 | artifacts.aggregate_boundary_lint |
| TASK-R04 | Evidence First（每个 Mutation 同事务写 Evidence） | §2.8 | T06 / T07 / T08 / T09 / T10 | T14-I03 / T15-P01 | artifacts.evidence_first / artifacts.evidence_record_per_mutation |
| TASK-R05 | Mutation 进入治理链（Transaction + Data + Evidence 阶段） | §2.7.1-§2.7.3 | T08 / T09 / T10 | T14-I01 | artifacts.mutation_governance_chain |
| TASK-R06 | 边由 Domain Event 驱动（非 Graph AI 推断） | §2.9.2 / §2.9.3 / §四 | T12 | T15-P03 / T15-P04 | artifacts.edge_by_domain_event / artifacts.has_child_edge_by_event |
| TASK-R09 | Physical Claim 由真实 Evidence 支撑 | §七、7.4 | T15 / T16 | T15-P06 / T15-P07 | artifacts.physical_evidence_json / artifacts.provenance_consistency |
| D-GATE-01 | Evidence append-only 四层防御 | §2.8.1 | T06 / T07 | T14-I03 | artifacts.evidence_append_only |
| D-GATE-05 | Graph 投影 ≤3s（Normal Mode） | §2.9 | T12 / T15 | T15-P05 | artifacts.projection_latency_le_3s |
| D-GATE-06 | 24 Entity Graph Contract（nodeType='Organization'） | §2.9.1 | T12 / T15 | T15-P02 | artifacts.graph_node_contract |

## 2.3 Design Mandatory RTM

| Design Mandatory | 内容 | Design 章节 | 落地 Task | 落地 Test ID | Evidence Item |
|---|---|---|---|---|---|
| Design Mandatory #1 | MoveOrganization 流程统一为 CommandHandler → Coordinator.MoveSubtree() | §三、3.1 | T03 / T10 | T13-U11 / T13-U16 | artifacts.move_via_coordinator / artifacts.no_move_method_on_aggregate |
| Design Mandatory #2 | OrganizationMoved 子树投影语义完整闭环（6 个子问题） | §三、3.2 | T07 / T12 / T15 | T15-P08 / T15-P09 / T15-P10 / T15-P11 | artifacts.subtree_projection_consistency / artifacts.event_replay_idempotent / artifacts.reconciliation / artifacts.has_child_edge_consistency |

## 2.4 spec.md 验收标准 RTM

| spec 验收项 | spec 章节 | 落地 Task | 落地 Test ID | Evidence Item |
|---|---|---|---|---|
| 组织创建后 Evidence Ledger 含对应 Evidence 记录 | §7.1 规则 1 | T06 / T08 | T14-I03 | artifacts.evidence_record_after_create |
| Outbox Event 发布至 Kafka 并投影至 Neo4j | §7.1 规则 2 | T08 / T09 / T10 / T12 | T14-I04 / T15-P02 | artifacts.outbox_publish_and_project |
| Neo4j Graph 含 Organization 节点 + BELONGS_TO + HAS_CHILD 边 | §7.1 规则 3 | T12 / T15 | T15-P02 / T15-P03 / T15-P04 | artifacts.neo4j_organization_node / artifacts.belongs_to_edge / artifacts.has_child_edge |
| 聚合根不变式校验通过 | §7.1 规则 4 | T02 / T03 / T13 | T13-U01 ~ T13-U16 | artifacts.aggregate_invariants |
| 组织树层级 >5 被拒绝 | §7.1 规则 5 | T02 / T03 / T13 / T14 | T13-U03 / T13-U13 / T14-I07 | artifacts.level_exceed_max_rejected |
| BELONGS_TO 边由 Domain Event 驱动 | §7.1 规则 6 | T12 / T15 | T15-P03 | artifacts.belongs_to_by_domain_event |
| HAS_CHILD 边由 Domain Event 驱动 | §7.1 规则 7 | T12 / T15 | T15-P04 | artifacts.has_child_by_domain_event |
| MoveOrganization 子树 level 递归更新 | §7.1 规则 8 | T10 / T14 | T14-I06 / T14-I08 | artifacts.subtree_level_recursive_update |
| Command Idempotency 幂等命中 | §7.1 规则 9 | T08 / T09 / T10 / T14 | T14-I11 | artifacts.command_idempotency |
| Unit Test 全部通过 | §7.2 规则 1 | T13 | T13-U01 ~ T13-U16 | artifacts.unit_test_pass |
| Integration Test 全部通过 | §7.2 规则 2 | T14 | T14-I01 ~ T14-I14 | artifacts.integration_test_pass |
| Physical Test 全部通过 | §7.2 规则 3 | T15 | T15-P01 ~ T15-P12 | artifacts.physical_test_pass |
| Test PASS ≠ Physical PASS | §7.2 规则 4 / §7.2 规则 5 | T15 / T16 | T15-P06 | artifacts.physical_evidence_json |
| Physical Evidence JSON 产出 | §7.3 规则 1 | T16 | T15-P06 | artifacts.physical_evidence_json |
| Provenance 一致性 | §7.3 规则 2 | T15 / T16 | T15-P07 | artifacts.provenance_consistency |
| TASK-R03 聚合根边界 | §7.4 规则 1 | T02 / T03 / T11 / T13 | T13-U07 | artifacts.aggregate_boundary_lint |
| TASK-R04 Evidence First | §7.4 规则 2 | T06 / T07 / T08 / T09 / T10 | T14-I03 | artifacts.evidence_first |
| TASK-R05 Mutation 进入治理链 | §7.4 规则 3 | T08 / T09 / T10 | T14-I01 | artifacts.mutation_governance_chain |
| TASK-R06 边由 Domain Event 驱动 | §7.4 规则 4 | T12 / T15 | T15-P03 / T15-P04 | artifacts.edge_by_domain_event |
| TASK-R09 Evidence First | §7.4 规则 5 | T15 / T16 | T15-P06 / T15-P07 | artifacts.physical_evidence_json |

## 2.5 RTM 覆盖率汇总

| 维度 | 总数 | 已覆盖 | 覆盖率 |
|---|---|---|---|
| 硬约束（R1.1-R1.4 + R2-R6） | 9 | 9 | 100% |
| 红线（TASK-R03/R04/R05/R06/R09 + D-GATE-01/05/06） | 8 | 8 | 100% |
| Design Mandatory（#1 / #2） | 2 | 2 | 100% |
| spec.md 验收标准 | 20 | 20 | 100% |
| Design 章节 | §二-§七 | §二-§七 | 100% |
| Task → Test → Evidence 追溯链 | 16 Task | 16 Task | 100% |

**RTM 完整性声明**：R1.1-R1.4 + R2-R6 + 8 项红线 + 2 项 Design Mandatory + 20 项 spec 验收标准全部有对应的 Task + Test + Evidence，无遗漏。

---

# 三、任务依赖图

```
T01（数据库迁移 V11）
  ↓
T02（OrganizationAggregate 领域模型）
  ↓
  ┌────────────┬────────────┬────────────┬────────────┐
  ↓            ↓            ↓            ↓            ↓
T03          T04          T05          T06
(Coordinator) (Repository) (UnitOfWork  (EvidenceWriter
              +GetSubtree   复用EV1-009) chain 1)
              +UpdateSubtree
              +CheckEnterprise)
  ↓            ↓            ↓            ↓
  │            │            │            ↓
  │            │            │          T07
  │            │            │          (StructuralWriter
  │            │            │           chain 2)
  │            │            │            ↓
  └────────────┴────────────┴────────────┘
                   ↓
         ┌─────────┴─────────┐
         ↓                   ↓
       T08                 T09
    (CreateHandler)     (UpdateHandler)
         ↓                   ↓
         └─────────┬─────────┘
                   ↓
                 T10
            (MoveHandler)
                   ↓
                 T11
            (CommandBus)
                   ↓
                 T12
          (Projection + HAS_CHILD
           + 子树投影 + Reconciliation)
                   ↓
         ┌─────────┴─────────┐
         ↓                   ↓
       T13                 T14
    (Unit Test)       (Integration Test)
         ↓                   ↓
         └─────────┬─────────┘
                   ↓
                 T15
            (Physical Test)
                   ↓
                 T16
         (Physical Evidence JSON)
```

**关键依赖说明**：
- T01 是基础设施层，必须最先执行（表结构不存在则其他任务无法验证）
- T02 是领域模型层，T03 / T04 / T06 依赖 T02 的聚合根与命令定义
- T03（Coordinator）依赖 T02（Aggregate），但 T03 不依赖 T04 / T05 / T06（Coordinator 在 T10 中才组装完整依赖）
- T07（StructuralWriter）依赖 T06（Writer 模式参考）
- T08 / T09 依赖 T02 / T04 / T05 / T06，可并行
- T10（MoveHandler）依赖 T03 / T04 / T05 / T06 / T07，是 Design Mandatory #1 核心
- T11（CommandBus）依赖 T08 / T09 / T10
- T12（Projection）依赖 T08 / T09 / T10（Outbox Event 已产出）
- T13 / T14 可并行（Unit Test 与 Integration Test 互不依赖，但都依赖 T11 / T12）
- T15 依赖 T13 / T14 全部 PASS
- T16 依赖 T15 PASS

---

# 四、测试分层汇总

| 测试层级 | 测试数量 | 测试范围 | 对应 Task | 对应 Test ID |
|---|---|---|---|---|
| **Unit Test** | 17 个 | 聚合根不变式校验、Coordinator 树操作逻辑、命令处理逻辑、事件产出、错误码映射、commandId 幂等命中、Design Mandatory #1（无 Move 方法）、R1.3（descendant version 不递增）、R1.2（Coordinator 不写业务属性） | T13 | T13-U01 ~ T13-U16 |
| **Integration Test** | 14 个 | 同事务原子写入、DB UNIQUE NULLS NOT DISTINCT 约束、Evidence Ledger 记录字段完整、Outbox 发布、CAS 并发冲突、MoveOrganization 子树递归更新、子树层级校验、子树 level 递归更新同事务原子、Evidence Hash Chain（2 条独立 chain）、Command Idempotency、RLS 租户隔离、2 条 Evidence 同事务原子、descendant version 不递增 | T14 | T14-I01 ~ T14-I14 |
| **Physical Test** | 12 个 | Evidence Hash Chain Physical 验证、Neo4j Organization 节点投影、BELONGS_TO 边投影、HAS_CHILD 边投影、投影最终一致 ≤3s + Neo4j 故障隔离、Physical Evidence JSON 产出、Provenance 一致性、**子树投影一致性**、**Event Replay 幂等性**、**Reconciliation**、**HAS_CHILD 边一致性**、**UNIQUE NULLS NOT DISTINCT Physical 验证** | T15 | T15-P01 ~ T15-P12 |
| **总计** | **43 个** | Unit 17 + Integration 14 + Physical 12 | T13 / T14 / T15 | — |

**测试通过标准**：
- Unit Test：全部 PASS（17 个）
- Integration Test：全部 PASS（14 个），仅 Mock 测试 PASS 不承认
- Physical Test：全部 PASS（12 个），Physical Evidence JSON 产出，仅 Unit + Integration PASS 不承认 Physical PASS，Mock PASS ≠ Real Infrastructure PASS

---

# 五、Evidence 要求

## 5.1 Physical Evidence JSON 字段要求

**路径**：`evidence/ev1/EBCX-EV1-010-organization-evidence.json`

**EV0 TASK-H07 14 个最小字段标准**（必须满足）：
1. `execution_id`：执行 ID（UUID v4）
2. `timestamp`：证据产出时间（ISO 8601）
3. `environment`：执行环境（"PostgreSQL 18.3 + Neo4j 5.x"）
4. `git_commit`：实际验证代码 commit hash
5. `test_command`：测试执行命令
6. `actual_output`：测试实际输出
7. `actual_metrics`：性能指标（Create/Update P95 ≤500ms、Move P95 ≤800ms、投影延迟 ≤3s）
8. `database_state`：数据库状态快照
9. `event_id`：领域事件 ID
10. `evidence_id`：Evidence 记录 ID
11. `trace_id`：链路追踪 ID
12. `failure_injection_result`：故障注入结果（Neo4j 故障隔离验证）
13. `verification_result`：验证结果（PASS / FAIL）
14. `verifier`：验证者

**EV1-010 扩展字段**：
- `taskId` / `taskName` / `gateName` / `infrastructure` / `testLayer` / `testNames` / `result` / `metrics` / `artifacts`
- `rtm`：Requirement Traceability Matrix
- `lockOrder`：事务内固定锁顺序声明（重点 ③）
- `pendingSemantics`：commandId pending 并发语义声明（重点 ①）
- `chainGenesis`：2 条 Chain 首条记录声明（重点 ②）
- `subtreeProjectionSemantics`：MoveOrganization 子树投影语义闭环声明（重点 ④）
- `hardConstraints`：R1.1-R1.4 + R2-R6 落地状态汇总
- `designMandatory`：Design Mandatory #1 / #2 落地状态汇总

## 5.2 git_commit provenance 要求

- Evidence Record 的 `Provenance.git_commit` 必须与实际验证代码 commit hash 一致
- Physical Evidence JSON 的 `git_commit` 必须与 Evidence Record 的 `git_commit` 一致
- **禁止破坏 Provenance**（spec §11 禁止事项 12）
- 代码变更时必须同步更新 git_commit，保持 Evidence 可溯源

---

# 六、硬约束覆盖情况

| 硬约束 | 覆盖任务 | 覆盖测试 | 覆盖状态 |
|---|---|---|---|
| R1.1 OrganizationAggregate Business State Mutation Root | T02 / T08 / T09 / T11 | T13-U01 / T13-U02 / T13-U08 / T14-I01 | ✅ 100% |
| R1.2 OrganizationTreeCoordinator Tree Structural State Mutation Authority | T03 / T10 / T11 | T13-U11 / T13-U15 / T14-I06 / T14-I14 | ✅ 100% |
| R1.3 descendant level 变更不经过 Aggregate Mutation，不递增 version | T03 / T10 / T14 | T13-U14 / T14-I14 | ✅ 100% |
| R1.4 MoveOrganization 产生 2 条 Evidence | T06 / T07 / T10 | T14-I13 / T15-P01 | ✅ 100% |
| R2 Idempotency + Organization + Evidence + Outbox 同 ACID 事务 | T01 / T05 / T08 / T09 / T10 | T14-I01 / T14-I09 | ✅ 100% |
| R3 Neo4j 不进入主事务 | T05 / T08 / T09 / T10 / T12 / T15 | T15-P05 | ✅ 100% |
| R4 Version CAS 乐观并发控制 | T04 / T09 / T10 | T14-I05 | ✅ 100% |
| R5 组织树层级 ≤5 不变式 | T02 / T03 / T08 / T10 | T13-U03 / T13-U13 / T14-I07 | ✅ 100% |
| R6 Organization 归属已存在 Enterprise，enterpriseId 不可变更 | T02 / T04 / T08 / T09 / T10 | T13-U02 / T13-U04 / T13-U10 | ✅ 100% |

**硬约束覆盖声明**：R1.1-R1.4 + R2-R6 全部 9 条硬约束 100% 覆盖，每条有对应的 Task + Test + Evidence。

---

# 七、禁止事项

> 以下禁止事项继承 spec.md v1.2 §11 + design.md v1.0 §文档定位 + 大G项目经理禁止事项，违反任一将导致 EV1-010-TASK Gate 直接 REJECT。

1. **禁止重新设计架构**（架构已 AUTHORIZED，本任务规划仅做"设计 → 施工任务"的 1:1 转换）
2. **禁止修改 EV1-010 spec.md v1.2**（已 FROZEN）
3. **禁止修改 EV1-010 design.md v1.0**（已 AUTHORIZED）
4. **禁止修改 EV0 spec.md / design.md / tasks.md**（均已 FROZEN）
5. **禁止修改 EV1-009 spec.md / design.md / tasks.md**（均已 FINAL CLOSED / FROZEN）
6. **禁止修改 EV1-009 `internal/enterprise/` 任何代码**（EV1-009 FROZEN）
7. **禁止修改 `business.enterprises` 表结构**（EV1-009 FROZEN）
8. **禁止修改 `EnterpriseCreated` / `EnterpriseUpdated` 领域事件**（EV1-009 FROZEN）
9. **禁止修改 `EBCX-ENTERPRISE-*` 错误码**（EV1-009 FROZEN）
10. **禁止修改 `evidence.evidence_ledger` 表结构**（EV1-003 FROZEN，仅扩展 aggregateType 枚举值）
11. **禁止扩大 EV1-010 Scope**（仅 Organization 聚合根 + OrganizationTreeCoordinator，不含 Person / MasterData / Permission / User / Authorization）
12. **禁止在 Task 阶段进入 Coding**（需 Task Gate PASS 后才授权）
13. **禁止自行进入 EV1-011 或任何后续任务**
14. **禁止 Neo4j 进入主事务**（R3）
15. **禁止非原子并发控制**（R4，禁止 SELECT-then-UPDATE 非原子方案）
16. **禁止 Mock 冒充 Physical Evidence**（TASK-R09）
17. **禁止 Test PASS 冒充 Physical PASS**（仅 Unit + Integration PASS 不承认 Physical PASS）
18. **禁止跨聚合直接调用**（TASK-R03，Organization 不直接调用 Enterprise 聚合根命令方法）
19. **禁止贫血模型**（OrganizationAggregate 必须封装不变式与命令处理）
20. **禁止破坏 Provenance**（Evidence.git_commit 必须与实际验证代码 commit 一致）
21. **禁止 EV1-010 自定义 Genesis Hash 规则**（必须复用 EV1-003 `GenesisHash(chainID)`，重点 ②.5）
22. **禁止跳过 `SELECT pg_advisory_xact_lock(hashtext($chainID))` 直接 `SELECT ... FOR UPDATE`**（空链并发不安全，重点 ②.5）
23. **禁止使用 session 级 `pg_advisory_lock`**（必须用事务级 `pg_advisory_xact_lock`，重点 ②.5）
24. **禁止跳过 SELECT ... FOR UPDATE 行锁直接 INSERT**（会破坏非空链并发排序确定性）
25. **禁止在 Coding 阶段调整事务内锁顺序**（本 tasks.md 重点 ③ 已写死，禁止锁顺序漂移）
26. **禁止引入后台清理 goroutine 处理 pending 超时**（归属 EV1-022 运维任务，重点 ①.2）
27. **禁止服务端自动轮询等待 pending 完成**（避免 goroutine 泄漏，重点 ①.1）
28. **禁止 OrganizationAggregate 含 Move() 方法**（Design Mandatory #1，Move 由 Coordinator 处理 R1.2）
29. **禁止 MoveOrganizationHandler 调用 OrganizationAggregate 的任何方法**（Design Mandatory #1，Handler 仅调用 Coordinator.MoveSubtree()）
30. **禁止 OrganizationMoved 事件含 affectedDescendantIds[]**（Design Mandatory #2 问题 1，重点 ④.1）
31. **禁止 Projection Consumer 使用相对值 SET**（`SET o.level = o.level + $delta`，重复 Replay 累加错误，重点 ④.3）
32. **禁止 HAS_CHILD 边直接 DELETE**（必须 validity='inactive' 软关闭，保留历史，重点 ④.4）
33. **禁止 2 条 chain 的 advisory lock 获取顺序不固定**（必须先 chain 1 后 chain 2，防止死锁，重点 ②.5）
34. **禁止跨 chain 复用 PreviousEvidenceHash**（每条 chain 独立，重点 ②.5）
35. **禁止 Coordinator 写 Organization 的 name / code**（R1.2，业务属性归 OrganizationAggregate R1.1）
36. **禁止 Coordinator 调用 descendant OrganizationAggregate 的 Mutation 方法**（R1.3，descendant level 变更通过直接 SQL UPDATE）
37. **禁止 descendant version 递增**（R1.3 / Q3-NORM，descendant level 变更不递增 version）
38. **禁止直接设置 level**（level 必须由聚合根/Coordinator 根据 parentId/newParentId 计算）
39. **禁止变更 enterpriseId**（R6，Organization 创建后 enterpriseId 不可变更）
40. **禁止组织树成环**（MoveOrganization 时必须校验组织树无环不变式）
41. **禁止组织树层级 >5**（R5，创建或移动组织至 level > 5 必须拒绝）

---

# 八、关键技术决策点（需大G项目经理 EV1-010-TASK Gate Review 审查）

> 以下关键技术决策点需大G项目经理在 EV1-010-TASK Gate Review 时审查裁决。**本任务规划未做任何超出 design.md v1.0 的新决策，仅做"设计 → 施工任务"的 1:1 转换。**

1. **4 项重点声明的写死确认**（本 tasks.md 重点 ① / ② / ③ / ④）：
   - 重点 ① Command Idempotency pending 并发语义：复用 EV1-009 模式，UNIQUE 冲突 → HTTP 409 EBCX-ORGANIZATION-IDEMPOTENCY-CONFLICT。请确认此写死语义符合预期。
   - 重点 ② 2 条 Evidence Chain 首条记录：复用 EV1-003 `GenesisHash(chainID)`，2 条独立 chain（organization-mutation-chain + organization-tree-structural-chain），advisory lock 顺序固定（先 chain 1 后 chain 2）。请确认 2 条 chain 独立性方案与 advisory lock 顺序符合预期。
   - 重点 ③ 事务内固定锁顺序：Create/Update 沿用 EV1-009 锁顺序，Move 新增子树批量 UPDATE + 2 条 chain lock。请确认 Move 锁顺序无死锁。
   - 重点 ④ MoveOrganization 子树投影语义闭环：6 个子问题的 Coding 阶段实现约束（事件不含 affectedDescendantIds[] / 第 2 条 Evidence 含 affectedDescendantIds[] / 绝对值 SET / HAS_CHILD 软关闭 / Reconciliation PG 为最终裁决源）。请确认 6 个子问题方案符合 Design Mandatory #2 要求。

2. **任务粒度确认**：本任务规划分解为 16 个主任务（T01-T16），无更深层级嵌套（Maximum 2 Levels），每个任务可独立验收。请确认任务粒度是否合适。

3. **任务依赖图确认**：本任务规划依赖图为 T01 → T02 → T03/T04/T05/T06/T07 → T08/T09/T10 → T11 → T12 → T13/T14 → T15 → T16。请确认依赖关系是否正确，无循环依赖，无遗漏依赖。

4. **RTM 覆盖率确认**：本任务规划 RTM 覆盖率 100%（9 硬约束 + 8 红线 + 2 Design Mandatory + 20 spec 验收标准 + 16 Task → Test → Evidence 追溯链）。请确认 RTM 无遗漏。

5. **Physical Evidence JSON 字段扩展确认**：本任务规划在 TASK-H07 最小字段标准基础上，额外增加 `rtm` / `lockOrder` / `pendingSemantics` / `chainGenesis` / `subtreeProjectionSemantics` / `hardConstraints` / `designMandatory` 七个字段。请确认此扩展是否符合 Physical Evidence 要求。

6. **Coding 授权前置条件确认**：本任务规划完成后，需大G项目经理进行 EV1-010-TASK Gate Review，裁决 PASS 后才授权 Coding。请确认 Gate Review 流程与 EV1-010-SPEC / EV1-010-DESIGN 一致，Task Gate PASS + Design Gate PASS 双授权后才进入 Coding。

7. **Design Mandatory #1 / #2 落地确认**：本任务规划 T03（Coordinator）+ T10（MoveHandler）落地 Design Mandatory #1，T07（StructuralWriter）+ T12（Projection）+ T15（Physical Test）落地 Design Mandatory #2。请确认 Design Mandatory 落地任务覆盖完整。

---

# 九、变更记录（Change Log）

## v1.0（2026-09-09，首次生成）

- **首次生成 EV1-010 Organization 聚合根编码任务规划 tasks.md v1.0**
- **基于 spec.md v1.2 FINAL PASS / CLOSED + design.md v1.0 AUTHORIZED**，严格遵守 Q1-Q4-NORM / R1.1-R1.4 / R2-R6 等规范性条款
- **分解为 16 个可执行、可验收、可追溯的施工任务**（T01-T16）：
  - T01 数据库迁移（V11__organization_aggregate.sql）
  - T02 OrganizationAggregate 领域模型（R1.1）
  - T03 OrganizationTreeCoordinator 领域服务（R1.2，Design Mandatory #1）
  - T04 OrganizationRepository 接口与 PostgreSQL 实现（R4 + GetSubtree + UpdateSubtreeLevels）
  - T05 UnitOfWork 事务边界（复用 EV1-009，R2）
  - T06 OrganizationEvidenceWriter（chain 1，R1.4）
  - T07 OrganizationTreeStructuralEvidenceWriter（chain 2，R1.4）
  - T08 CreateOrganizationHandler（R1.1 / R2 / R6）
  - T09 UpdateOrganizationHandler（R1.1 / R2 / R4）
  - T10 MoveOrganizationHandler（R1.2 / R2 / R4 / R5，Design Mandatory #1 核心）
  - T11 OrganizationCommandBus
  - T12 Organization 投影规则 + HAS_CHILD + 子树投影 + Reconciliation（R3，Design Mandatory #2）
  - T13 Unit Test（17 个）
  - T14 Integration Test（14 个）
  - T15 Physical Test（12 个）
  - T16 Physical Evidence JSON 产出
- **覆盖 4 项重点声明**：
  - 重点 ① Command Idempotency pending 并发语义（复用 EV1-009 模式）
  - 重点 ② 2 条 Evidence Chain 首条记录（2 条独立 chain + advisory lock 顺序固定）
  - 重点 ③ 事务内固定锁顺序（Create/Update/Move 三种锁顺序，Move 含子树批量 UPDATE + 2 条 chain lock）
  - 重点 ④ MoveOrganization 子树投影语义闭环（Design Mandatory #2 的 6 个子问题 Coding 阶段约束）
- **覆盖 9 条硬约束**：R1.1 / R1.2 / R1.3 / R1.4 / R2 / R3 / R4 / R5 / R6 全部 100% 覆盖
- **覆盖 2 项 Design Mandatory**：#1（MoveOrganization 流程统一）+ #2（子树投影语义完整闭环）全部 100% 覆盖
- **建立 RTM 矩阵**：9 硬约束 + 8 红线 + 2 Design Mandatory + 20 spec 验收标准，覆盖率 100%
- **测试总计**：Unit Test 17 个 + Integration Test 14 个 + Physical Test 12 个 = 43 个测试
- **声明与 EV1-009 / EV0 的兼容性**：复用基础设施 + 兼容性扩展枚举值 + 不修改 EV1-009 / EV0 任何冻结产物

---

> **文档结束**
> 本 tasks.md 定义 EBCX-EV1-010 Organization 聚合根的完整编码任务规划，覆盖：
> - 4 项重点声明（pending 并发语义 / 2 条 Chain 首条记录 / 事务内固定锁顺序 / MoveOrganization 子树投影语义闭环）
> - 16 个施工任务（T01-T16，含数据库迁移 / 领域模型 / Coordinator 领域服务 / 持久化 / 事务边界 / 2 个 Evidence 适配 / 3 个 Handler / 命令分发 / 投影规则 + HAS_CHILD + 子树投影 + Reconciliation / Unit Test / Integration Test / Physical Test / Physical Evidence JSON）
> - RTM 矩阵（9 硬约束 + 8 红线 + 2 Design Mandatory + 20 spec 验收标准，覆盖率 100%）
> - 任务依赖图
> - 硬约束覆盖情况（R1.1-R1.4 + R2-R6 全部 100% 覆盖）
> - 测试分层汇总（Unit 17 + Integration 14 + Physical 12 = 43 个测试）
> - Evidence 要求（Physical Evidence JSON 14 个最小字段 + git_commit provenance）
> - 禁止事项（41 项，含不修改 EV1-009 / EV0 冻结产物）
> - 关键技术决策点（7 项）
>
> **9 条硬约束 + 2 项 Design Mandatory 落地状态**：R1.1 ✅ / R1.2 ✅ / R1.3 ✅ / R1.4 ✅ / R2 ✅ / R3 ✅ / R4 ✅ / R5 ✅ / R6 ✅ / Design Mandatory #1 ✅ / Design Mandatory #2 ✅ 全部对齐，每项有对应的 Task + Test + Evidence。
>
> **未修改声明**：本任务规划（v1.0）未修改 spec.md v1.2（FROZEN）、未修改 design.md v1.0（AUTHORIZED）、未修改 EV0 文档（FROZEN）、未修改 EV1-009 文档与代码（FROZEN）、未扩大 Scope（仅 Organization 聚合根 + OrganizationTreeCoordinator）、未进入 Coding（Task Gate 未授权）。
>
> **下一步**：提交大G项目经理进行 EV1-010-TASK Gate Review，裁决通过后方可授权 Coding（由 coding-agent 执行）。