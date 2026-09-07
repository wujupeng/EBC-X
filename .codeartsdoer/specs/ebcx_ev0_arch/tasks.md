# EBC-X EV0 编码任务规划文档（tasks.md）

> **项目：EBC-X — Enterprise Business & Industrial Operating System（企业与工业智能运营操作系统）**
> **阶段：EV0 — Architecture & Policy Alignment → 编码任务规划（tasks.md）**
> **文档版本：v1.0（首版，基于 spec.md v1.1 + design.md v1.1 冻结基线生成）**
> **状态：🟡 TASKS v1.0（待大G项目经理体系 EV0-TASKS Gate 复审）**
> **需求基线：`.codeartsdoer/specs/ebcx_ev0_arch/spec.md` v1.1（EV0-SPEC PASS / CLOSED / 🔒 FROZEN，不可变基线）**
> **设计基线：`.codeartsdoer/specs/ebcx_ev0_arch/design.md` v1.1（EV0-DESIGN PASS / CLOSED / 🔒 FROZEN，不可变基线）**
> **产品/架构总设计：大G项目经理体系**
> **核心工程化：华为云团队**
> **全球交付与产品主权：HTKIS**

---

## 文档定位与约束声明

本文档是 spec.md（v1.1，FROZEN）+ design.md（v1.1，FROZEN）的**工程化、可执行化、可验证化**展开，回答：

> **"EBC-X 究竟怎样把已冻结的 24 项设计决策（D01~D24）+ 8 项 Architecture Hardening（D-GATE-01~08）转化为可执行、可测试、可验证、可产生 Physical Evidence 的工程任务。"**

**不可变基线约束**：
- ❌ 禁止修改 spec.md v1.1（已冻结）
- ❌ 禁止修改 design.md v1.1（已冻结）
- ❌ 禁止在 Tasks 阶段重新发明架构（TASK-R01）
- ❌ 禁止把 14 Modules 拆成大量微服务（TASK-R02）
- ✅ 只允许 Design → Implementation Tasks 转化，严格遵循 10 条施工红线（TASK-R01~R10）

**覆盖范围**：EV1~EV12 全部阶段任务，重点细化 EV1 Enterprise Core 与 EV2 Transaction Core（第一阶段可执行任务），EV3~EV12 为里程碑级任务。

---

## 任务规划原则

1. **垂直切割**：按业务功能/基础设施能力分组，而非按技术层次分组
2. **Evidence First**：每个关键任务含 Unit Test → Integration Test → Physical Verification → Evidence → Gate
3. **可验收**：每个任务必须有明确的完成标准与 Physical Evidence 产出
4. **原子性**：一个任务只做一件事，EV1/EV2 任务粒度控制在 1~3 人日
5. **有序性**：任务按依赖关系排序，形成可执行 DAG，被依赖者在前
6. **不耦合**：聚合根间通过 Domain Event + Orchestrator 编排，禁止跨聚合直接调用（TASK-R03）
7. **不假完成**：每个任务必须能产生 Physical Evidence，禁止"看似完成但无法验证"

---

## 任务编号规范

- **格式**：`EBCX-EV{阶段号}-{任务序号}`，如 `EBCX-EV1-001`
- **阶段号**：1~12，对应 EV1~EV12
- **任务序号**：001~999，按依赖顺序递增

---

## 任务依赖 DAG（高层视图）

```text
EV1 Enterprise Core（基础设施 + 聚合根）
  ├─ EBCX-EV1-001 项目脚手架
  ├─ EBCX-EV1-002 PostgreSQL schema 基础
  ├─ EBCX-EV1-003 Evidence Ledger 基础设施（D-GATE-01 四层防御）  ← TASK-R04 从第一批代码起
  ├─ EBCX-EV1-004 Outbox + EventBus 基础设施
  ├─ EBCX-EV1-005 Neo4j Graph Projection 基础设施（TASK-R06 Projection）
  ├─ EBCX-EV1-006 Object Storage Artifact 基础设施
  ├─ EBCX-EV1-007 HTKIS-AF 安全基座（OAuth2+JWT+RLS+审计）
  ├─ EBCX-EV1-008 多租户基础（RLS + Pack 路由）
  ├─ EBCX-EV1-009~013 Enterprise/Organization/Person/MasterData/Permission 聚合根
  ├─ EBCX-EV1-014 24 Entity Graph Schema + Node/Edge Contract（D-GATE-06）
  ├─ EBCX-EV1-015 Observability 基础
  ├─ EBCX-EV1-016 Benchmark Harness 框架（TASK-R08 B1 Profile）
  ├─ EBCX-EV1-017 REST API /api/v1/rel/* 第一入口
  ├─ EBCX-EV1-018 GraphQL 适配层
  ├─ EBCX-EV1-019 Event Contract + Schema Registry
  ├─ EBCX-EV1-020 CQRS Command/Query 分离基础
  ├─ EBCX-EV1-021 DevSecOps CI/CD 流水线
  ├─ EBCX-EV1-022 IaC 基础设施（Terraform + 华为云）
  ├─ EBCX-EV1-023 B1 Benchmark 首次执行 + Measured Baseline
  └─ EBCX-EV1-024 EV1 Gate 评审准备
        ↓（解锁 EV2）
EV2 Transaction Core（聚合根 + Orchestrator + 治理）
  ├─ EBCX-EV2-001~004 Order/Contract/Invoice/Payment 聚合根
  ├─ EBCX-EV2-005 Transaction Orchestrator 11 阶段编排（D-GATE-02）
  ├─ EBCX-EV2-006 Saga/Workflow 跨服务编排 + Compensation
  ├─ EBCX-EV2-007 Domain Event 契约 + Schema Registry
  ├─ EBCX-EV2-008 CQRS Read Model 投影
  ├─ EBCX-EV2-009 Evidence Provenance & Data Lineage
  ├─ EBCX-EV2-010 Policy Engine 基础
  ├─ EBCX-EV2-011 Approval 聚合根（一等公民）
  ├─ EBCX-EV2-012 Agent Runtime 基础（三重治理）
  ├─ EBCX-EV2-013 Agent Execution Authorization 模型（D-GATE-07）
  ├─ EBCX-EV2-014 REST API Transaction/Evidence/Policy/Agent/Graph
  ├─ EBCX-EV2-015 B2~B5 Profile 预留框架
  ├─ EBCX-EV2-016 B1 Benchmark 执行 + Measured Baseline
  └─ EBCX-EV2-017 EV2 Gate 评审准备
        ↓（解锁 EV3~EV12，里程碑级）
EV3~EV12 里程碑级任务（10 个）
        ↓
集成测试 / 部署配置 / 评审验证 任务组
        ↓
EV0-TASKS Gate 评审
```

---

# 一、EV1 Enterprise Core 细化任务

> **目标**：建立 Core + Evidence + Data + Policy 地基（spec.md §5.12.1 红线二），完成企业/组织/主数据/权限聚合根 + 全部横切基础设施。
> **关联 spec.md**：§5.10 EV1、§5.2 模块树、§5.3 技术栈、§5.4 数据库与事件模型、§5.5 Evidence Graph、§5.7 权限安全、§5.8 全球化、§5.9 华为云部署
> **关联 design.md**：D01 Logical Architecture、D02 Domain Boundary、D03 Enterprise Core、D05 Evidence Ledger、D06 Graph Projection、D07 Outbox+EventBus、D13 Security、D14 Multi-Tenant、D15 Data Architecture、D16 API、D17 Event Contract、D22 Observability、D23 Audit、D24 Deployment、D-GATE-01/06/08

## 1.1 项目脚手架与基础设施

### EBCX-EV1-001：搭建 Modular Monolith 项目脚手架（Go + Rust + React/Antd/Vite + 华为云 CCE）
- **关联 spec.md**：§5.3.1 规则 1/6/7（云原生 + 双语言 + 前端）、§5.12.1 红线三（禁止数百微服务）
- **关联 design.md**：D01 Logical Architecture、D24 Deployment Architecture、§2.5.1 Design Freeze Principle #1
- **实现内容**：
  - 建立 Go 主业务服务仓库骨架（14 模块 Go package 边界：enterprise/transaction/finance/scm/manufacturing/crm/project/eam/quality/datafabric/evidence/policy/agent/digitaltwin + platform 横切）
  - 建立 Rust 高性能/安全关键组件仓库骨架（Evidence hash 计算、加密、Schema 校验等）
  - 建立 React + Antd + Vite v6.4.3 前端仓库骨架（管理面 + 工程化 UI）
  - 建立 Go ↔ Rust FFI / gRPC 接口边界（明确边界，禁止无边界混用）
  - 模块边界编译期校验脚本（Go package lint + Rust crate lint）
- **验收标准**：
  - [ ] 14 模块 Go package 边界编译期可校验，跨聚合直接调用被 lint 拒绝
  - [ ] Go + Rust 双语言构建脚本可独立构建 + 联合构建
  - [ ] React + Antd + Vite 前端可独立构建并产出 dist/
  - [ ] 模块间调用 ≤5ms（同进程基准测试通过）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-001-scaffold-evidence.json`（含 14 模块清单、构建日志、lint 通过截图、模块间调用延迟基准测试报告）
- **依赖任务**：无（首任务）
- **遵守红线**：TASK-R01（不重新设计架构）、TASK-R02（不拆大量微服务）

### EBCX-EV1-002：建立 PostgreSQL schema 基础（7 个 schema 划分 + 迁移工具）
- **关联 spec.md**：§5.3.1 规则 2（PostgreSQL 持久层）、§5.4 数据库与事件模型
- **关联 design.md**：D15 Data Architecture（schema 划分：business/evidence/outbox/audit/master_data/policy/tenant）
- **实现内容**：
  - 创建 7 个 PostgreSQL schema：`business` / `evidence` / `outbox` / `audit` / `master_data` / `policy` / `tenant`
  - 集成 Flyway/Liquibase 迁移工具，迁移脚本版本化管理
  - 建立 Migration Role（DDL + DML 全权限，仅迁移窗口期激活）与 Runtime Role（仅 INSERT + SELECT）的权限分离初版
  - 建立 schema 迁移 CI 校验（禁止手工变更生产 schema）
- **验收标准**：
  - [ ] 7 个 schema 全部创建并通过迁移测试
  - [ ] Flyway/Liquibase 迁移脚本可重复执行且幂等
  - [ ] Migration Role 与 Runtime Role 权限分离校验通过
  - [ ] 迁移脚本版本化纳入 git 管理
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-002-schema-evidence.json`（含 7 schema DDL、迁移日志、权限分离校验报告）
- **依赖任务**：EBCX-EV1-001
- **遵守红线**：TASK-R04（Evidence Ledger 从第一批代码起）

### EBCX-EV1-003：实现 Evidence Ledger 基础设施（append-only + 四层纵深防御 + RLS + Runtime Role）🔴 关键
- **关联 spec.md**：§4.2 规则 2（不可篡改）、§5.4.1 规则 2/3（append-only + 审计）、§6.1（数据约束）、§5.7.1 规则 2/4（RLS + 审计不可篡改）
- **关联 design.md**：D05 Evidence Ledger、D-GATE-01（Evidence Ledger 不可篡改安全边界四层纵深防御）
- **实现内容**：
  - **第一层（DB 权限模型）**：创建 `ebcx_runtime_role`（仅 `INSERT + SELECT ON evidence.*`，`REVOKE UPDATE, DELETE, TRUNCATE`）+ `ebcx_migration_role`（DDL + DML，仅迁移窗口期）+ `ebcx_audit_role`（仅 INSERT + SELECT on audit.*）+ DB Owner（不用于运行时）
  - **第二层（DB 触发器/RULE）**：`CREATE RULE evidence_no_update AS ON UPDATE TO evidence.evidence_ledger DO INSTEAD NOTHING;` + 同 ON DELETE + 同 ON TRUNCATE
  - **第三层（应用层校验）**：Repository 层禁止 emit UPDATE/DELETE 语句，代码评审 lint 规则
  - **第四层（纵深）**：审计表 `evidence.evidence_audit`（append-only，记录所有 evidence 表访问）+ legal_hold 字段 + 定期 hash 链校验 job
  - **Evidence 必含字段**：evidence_id / evidence_type / payload / evidence_hash（SHA-256 of payload+source_event_id+transaction_id）/ source_event_id / transaction_id / tenant_id / created_at / created_by / version / provenance（JSONB）/ correlation_id / causation_id / legal_hold
  - **修改语义**：修改 = 新版本 Evidence（correlation_id + causation_id 串联）；删除 = tombstone Evidence（物理保留）
  - **hash 链校验**：每日 job 校验 `evidence_hash = SHA256(payload || source_event_id || transaction_id)`，不匹配则告警 + 审计 + 阻断决策
  - **RLS 策略**：`CREATE POLICY evidence_tenant_isolation ON evidence.evidence_ledger USING (tenant_id = current_setting('app.tenant_id')::uuid);`
- **验收标准**：
  - [ ] Runtime Role 尝试 UPDATE/DELETE/TRUNCATE evidence 表被数据库拒绝（Physical Verification：执行 SQL 验证返回权限错误）
  - [ ] DB 触发器/RULE 拒绝 UPDATE/DELETE 并写入审计
  - [ ] 应用层 Repository 无 UPDATE/DELETE 语句（lint 通过）
  - [ ] hash 链校验 job 检测到篡改时告警 + 审计 + 阻断
  - [ ] legal_hold = TRUE 时 retention job 不得清理
  - [ ] 修改语义验证：写入新版本 Evidence 后原 Evidence 保留不动
  - [ ] RLS 跨租户访问被拦截并审计
  - [ ] Evidence 写入吞吐 ≥5000 EPS（基准测试）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-003-evidence-ledger-evidence.json`（含四层防御校验报告、权限拒绝日志、hash 链校验报告、吞吐基准测试、RLS 拦截日志）
- **依赖任务**：EBCX-EV1-002
- **遵守红线**：TASK-R04（Evidence Ledger 从第一批代码起，四层纵深防御 + Runtime Role 无 UPDATE/DELETE + append-only + hash 链 + correlation_id/causation_id 一并落地）、TASK-R09（Evidence First）

### EBCX-EV1-004：实现 Outbox + EventBus 基础设施（DMS/Kafka + 至少一次 + 幂等）
- **关联 spec.md**：§4.2 规则 4（Outbox + EventBus 异步投影）、§5.4.2（交互流程）
- **关联 design.md**：D07 Outbox + EventBus、§2.5.3 Outbox + EventBus 异步投影
- **实现内容**：
  - 创建 `outbox.outbox_events` 表（event_id PK / event_type / event_version / aggregate_id / aggregate_version / tenant_id / trace_id / payload JSONB / evidence_ref / occurred_at / published_at NULL）
  - 实现 Outbox Publisher（轮询 `published_at IS NULL` 的未投递事件，发布至华为云 DMS Kafka）
  - 实现幂等消费者（基于 event_id 去重，消费者维护已处理 event_id 表）
  - 实现至少一次投递语义（重投间隔指数退避 1s/2s/4s/8s，最大收敛 3s）
  - 实现 DLQ（Dead Letter Queue）：重投 N=5 次后仍失败进入 DLQ，DLQ 不阻断 Transaction Core 但阻断对应 Event 的 Graph 投影
  - 集成华为云 DMS（Kafka 兼容），保持可迁移性不产生不可逆厂商锁定
- **验收标准**：
  - [ ] Outbox 与业务数据同事务原子写入（PostgreSQL 事务提交即业务完成，RPO≤0）
  - [ ] 至少一次投递验证：模拟消费者崩溃后恢复，事件不丢失
  - [ ] 幂等消费者验证：重复投递同一 event_id 返回同一结果
  - [ ] DLQ 验证：重投 5 次后进入 DLQ，Transaction Core 不受影响
  - [ ] 收敛 ≤3s（B1 Normal Mode 基准测试）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-004-outbox-evidence.json`（含同事务原子性测试、至少一次投递测试、幂等测试、DLQ 测试、收敛延迟基准报告）
- **依赖任务**：EBCX-EV1-003
- **遵守红线**：TASK-R05（Mutation 进入 Outbox 治理链）、TASK-R09（Evidence First）

### EBCX-EV1-005：实现 Neo4j Graph Projection 基础设施（异步投影 + 重建 + DLQ + 模式分级）🔴 关键
- **关联 spec.md**：§4.2 规则 4（Neo4j 失败不拖垮 Transaction Core）、§5.5 Evidence Graph、§5.4.3（异常场景）
- **关联 design.md**：D06 Evidence Graph Projection、D19 Failure/Recovery Model、D-GATE-05（Graph Projection ≤3s 适用条件与模式分级）
- **实现内容**：
  - 创建 Neo4j schema：24 类节点标签 + 8 类边类型 + 索引 on nodeId/tenantId/sourceEvidenceId + 约束 on nodeId 唯一
  - 实现 Graph Projection Consumer：消费 Outbox Event → 转换为 Cypher MERGE 语句 → 异步执行
  - 实现幂等投影（基于 event_id 去重）
  - 实现重建策略：Neo4j 全量重建 = 清空 → 从 PostgreSQL Evidence Ledger 全量 Event Replay → 重新投影
  - 实现模式分级：Normal Mode（P95 ≤3s）/ Degraded Mode（lag 监控 + 告警）/ Recovery Mode（重投收敛）/ Rebuild Mode（查询降级至 PG 直查 + 告警）
  - 实现 6 项关键指标采集：event_lag / projection_lag / consumer_lag / rebuild_duration / failed_projection_count / dlq_count
  - 实现查询降级：Neo4j 故障时 Graph 查询降级至 PostgreSQL 直查 + 告警
  - **禁止业务代码直接把 Neo4j 当 Source of Truth**（Neo4j 只通过 Outbox+EventBus 异步投影写入）
- **验收标准**：
  - [ ] Neo4j 故障时 Transaction Core 不受影响（Physical Verification：停止 Neo4j，发起交易仍成功）
  - [ ] Outbox Event 积压待 Neo4j 恢复后重投成功
  - [ ] 全量重建 RTO ≤30min
  - [ ] Normal Mode P95 projection_lag ≤3s（B1 基准测试）
  - [ ] 6 项指标全部接入华为云 APM/云监控
  - [ ] DLQ 消息标记为 `projection_failed`，不阻断 Transaction
  - [ ] 业务代码无直接 Neo4j 写入（lint 通过，仅 Projection Consumer 写入）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-005-graph-projection-evidence.json`（含 Neo4j 故障隔离测试、重建测试、模式分级测试、6 项指标监控截图、lint 报告）
- **依赖任务**：EBCX-EV1-004
- **遵守红线**：TASK-R06（Neo4j 必须是 Projection，禁止业务代码直接写 Neo4j）、TASK-R09（Evidence First）

### EBCX-EV1-006：实现 Object Storage Artifact 基础设施（OBS + WORM + 命名规范）
- **关联 spec.md**：§5.3.1 规则 3（三层真相模型 Artifact Truth）、§5.4（Artifact Store）
- **关联 design.md**：D15 Data Architecture（OBS 命名规范 + WORM）、§2.5.2 三层真相模型
- **实现内容**：
  - 集成华为云 OBS（对象存储），保持可迁移性不产生不可逆厂商锁定
  - 实现命名规范：`{tenantId}/{evidenceId}/{artifactType}/{version}/{filename}`
  - 实现 WORM（Write Once Read Many）或等价不可变保护（OBS WORM 或版本化 + 删除保护）
  - 实现 Artifact 上传/下载接口（受 RLS + 权限约束）
  - 实现 Artifact 引用记录至 Evidence Ledger（artifact_ref 字段）
- **验收标准**：
  - [ ] Artifact 上传后不可修改、不可删除（WORM 验证）
  - [ ] 命名规范校验通过（所有 Artifact 路径符合规范）
  - [ ] Artifact 引用可追溯至 Evidence Ledger
  - [ ] 跨租户 Artifact 访问被拦截
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-006-obs-evidence.json`（含 WORM 验证、命名规范校验、权限拦截日志）
- **依赖任务**：EBCX-EV1-003
- **遵守红线**：TASK-R09（Evidence First）

### EBCX-EV1-007：实现 HTKIS-AF 安全基座（OAuth2 + JWT + RLS + 审计 append-only）
- **关联 spec.md**：§4.3（安全性）、§5.7（权限安全全模块）、§5.12.1 红线五
- **关联 design.md**：D13 Security Architecture（HTKIS-AF 归并 + OAuth2+JWT+RLS+审计 append-only）
- **实现内容**：
  - 归并 HTKIS-AF 作为唯一安全基座（禁止各模块自建安全能力，lint 校验）
  - 实现 OAuth2 + JWT 签发/校验（AuthGateway 组件）
  - 实现 PostgreSQL RLS 策略（tenant_id + org_id 维度，所有业务表启用 RLS）
  - 实现 append-only 审计日志（`audit.audit_log` 表，Runtime Role 无 UPDATE/DELETE）
  - 实现密钥管理服务（KMS，字段级加密密钥轮换）
  - 实现 TLS 1.2+ 强制（TLSGateway 组件）
  - 实现敏感数据字段级加密（凭证/PII 传输 + 存储双重加密）
- **验收标准**：
  - [ ] 未携带有效 JWT 的请求返回 401 并审计
  - [ ] RLS 越权访问被拦截并返回空集或 403（EBCX-TENANT-ISOLATION）
  - [ ] 审计日志 append-only 强制（UPDATE/DELETE 被拒绝）
  - [ ] 敏感数据传输 + 存储均为密文（抓包验证 + 存储 inspection）
  - [ ] JWT 校验 ≤10ms、RLS 行级过滤 ≤5ms
  - [ ] 各模块无自建安全能力（lint 通过）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-007-security-evidence.json`（含 401/403 测试、RLS 拦截日志、审计 append-only 验证、加密验证、性能基准）
- **依赖任务**：EBCX-EV1-002、EBCX-EV1-003
- **遵守红线**：TASK-R04（审计 append-only）、TASK-R09（Evidence First）

### EBCX-EV1-008：实现多租户基础（租户隔离 + RLS + Pack 路由）
- **关联 spec.md**：§5.7.1 规则 5（租户隔离）、§5.8（Global Core + Country Pack）
- **关联 design.md**：D14 Multi-Tenant Architecture（共享 schema + RLS + Pack 路由 + 核心表不分叉）
- **实现内容**：
  - 实现共享 schema + 行级隔离（RLS）模型，每表含 tenant_id 列
  - 实现 RLS 策略：`current_setting('app.tenant_id') = tenant_id`
  - 实现 Pack 路由：请求头 `X-Tenant-Id` + `X-Country-Pack` 路由至对应 Country Pack 扩展表
  - 预留 5 个 Country Pack 扩展点：China / EU / US / Japan / ASEAN
  - 实现核心表不分叉校验（编译期/迁移期校验核心表结构跨 Country Pack 一致）
  - 实现本地化字段以扩展表附加（`{table}_{country}_ext`）
- **验收标准**：
  - [ ] 跨租户访问被 RLS 拦截并审计（Physical Verification）
  - [ ] 5 个 Country Pack 扩展点全部预留
  - [ ] 核心表结构跨 Country Pack 一致（编译期校验通过）
  - [ ] 本地化字段以扩展表附加（无核心表分叉）
  - [ ] RLS 行级过滤 ≤5ms
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-008-multi-tenant-evidence.json`（含 RLS 拦截测试、5 Country Pack 扩展点清单、核心表不分叉校验报告）
- **依赖任务**：EBCX-EV1-007
- **遵守红线**：TASK-R01（不重新设计架构）、TASK-R09（Evidence First）

## 1.2 Enterprise Core 聚合根

### EBCX-EV1-009：实现 Enterprise 聚合根
- **关联 spec.md**：§5.10 EV1 Enterprise Core、§5.2 模块树
- **关联 design.md**：D03 Enterprise Core（EnterpriseAggregate）、D02 Domain Boundary
- **实现内容**：
  - 实现 `EnterpriseAggregate`（企业根节点，含 enterpriseId / name / version / sourceEvidenceId / tenantId）
  - 实现聚合根不变式（企业唯一性、版本单调递增）
  - 实现命令处理：CreateEnterprise / UpdateEnterprise（新版本）
  - 实现事件：EnterpriseCreated / EnterpriseUpdated
  - 写入 Evidence Ledger + Outbox Event（同事务原子）
  - 投影至 Neo4j Graph（Enterprise 节点）
- **验收标准**：
  - [ ] Enterprise 创建后 Evidence Ledger 含对应 Evidence 记录
  - [ ] Outbox Event 发布至 Kafka 并投影至 Neo4j
  - [ ] Neo4j Graph 含 Enterprise 节点（nodeType='Enterprise'）
  - [ ] 聚合根不变式校验通过
  - [ ] Unit Test + Integration Test 全部通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-009-enterprise-evidence.json`（含聚合根测试、Evidence 记录、Graph 节点验证）
- **依赖任务**：EBCX-EV1-003、EBCX-EV1-004、EBCX-EV1-005、EBCX-EV1-014
- **遵守红线**：TASK-R03（聚合根边界）、TASK-R04（Evidence First）、TASK-R05（Mutation 进入治理链）、TASK-R09（Evidence First）

### EBCX-EV1-010：实现 Organization 聚合根（组织树层级 ≤5）
- **关联 spec.md**：§5.10 EV1 Enterprise Core
- **关联 design.md**：D03 Enterprise Core（OrganizationAggregate，组织树层级 ≤5）
- **实现内容**：
  - 实现 `OrganizationAggregate`（含 orgId / enterpriseId / parentId / level / version / sourceEvidenceId）
  - 实现组织树层级 ≤5 校验（parentId 自引用，创建时校验 parent.level + 1 ≤ 5）
  - 实现命令处理：CreateOrganization / UpdateOrganization / MoveOrganization
  - 实现事件：OrganizationCreated / OrganizationUpdated / OrganizationMoved
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Organization 节点 + BELONGS_TO 边至 Enterprise）
- **验收标准**：
  - [ ] 组织树层级 >5 被拒绝（不变式校验）
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] BELONGS_TO 边由 Domain Event 驱动创建（非 Graph AI 推断）
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-010-organization-evidence.json`
- **依赖任务**：EBCX-EV1-009
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R06（边由 Domain Event 驱动）、TASK-R09

### EBCX-EV1-011：实现 Person 聚合根（人员 + 角色）
- **关联 spec.md**：§5.10 EV1 Enterprise Core
- **关联 design.md**：D03 Enterprise Core（PersonAggregate）
- **实现内容**：
  - 实现 `PersonAggregate`（含 personId / orgId / roles / version / sourceEvidenceId）
  - 实现命令处理：CreatePerson / AssignRole / UpdatePerson
  - 实现事件：PersonCreated / RoleAssigned / PersonUpdated
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Person 节点 + BELONGS_TO 边至 Organization）
- **验收标准**：
  - [ ] Person 创建后关联至 Organization（BELONGS_TO 边）
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-011-person-evidence.json`
- **依赖任务**：EBCX-EV1-010
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R09

### EBCX-EV1-012：实现 MasterData 聚合根（产品/物料/客户/供应商等主数据）
- **关联 spec.md**：§5.10 EV1 Enterprise Core、§5.2 模块树
- **关联 design.md**：D03 Enterprise Core（MasterDataAggregate）
- **实现内容**：
  - 实现 `MasterDataAggregate`（承载 Product / Material / Customer / Supplier 等主数据）
  - 实现命令处理：CreateProduct / CreateMaterial / CreateCustomer / CreateSupplier / UpdateMasterData
  - 实现事件：ProductCreated / MaterialCreated / CustomerCreated / SupplierCreated / MasterDataUpdated
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Product/Material/Customer/Supplier 节点）
- **验收标准**：
  - [ ] 4 类主数据全部实现并投影至 Graph
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-012-master-data-evidence.json`
- **依赖任务**：EBCX-EV1-009
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R09

### EBCX-EV1-013：实现 Permission 聚合根（权限 + 角色 + RLS 上下文）
- **关联 spec.md**：§5.10 EV1 Enterprise Core、§5.7 权限安全
- **关联 design.md**：D03 Enterprise Core（PermissionAggregate，对接 HTKIS-AF）
- **实现内容**：
  - 实现 `PermissionAggregate`（含角色 + 权限 + RLS 上下文）
  - 实现权限继承单调校验
  - 对接 HTKIS-AF 统一供给（禁止自建，lint 校验）
  - 实现命令处理：CreateRole / AssignPermission / AssignRoleToPerson
  - 实现事件：RoleCreated / PermissionAssigned / RoleAssignedToPerson
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Policy 节点 + GOVERNED_BY 边）
- **验收标准**：
  - [ ] 权限继承单调校验通过
  - [ ] Permission 对接 HTKIS-AF（无自建安全能力）
  - [ ] 权限校验 ≤10ms
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-013-permission-evidence.json`
- **依赖任务**：EBCX-EV1-007、EBCX-EV1-011
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R07（Agent Execution 经 Policy/Authorization）、TASK-R09

## 1.3 Graph Schema 与横切基础设施

### EBCX-EV1-014：实现 24 Entity Graph Schema + Node/Edge Contract（D-GATE-06）🔴 关键
- **关联 spec.md**：§5.5（Evidence Graph 24 类节点 + 8 类边 + Approval 一等公民）、§6.2/6.3（数据约束）
- **关联 design.md**：D06 Evidence Graph Projection、D-GATE-06（24 Entity + 8 Edge Canonical Graph Contract）
- **实现内容**：
  - **24 Node Type 枚举锁定**：Enterprise / Organization / Person / Product / Material / Supplier / Customer / Order / Contract / Invoice / Payment / Production / Quality / Asset / Project / Patent / R&D / Data / Evidence / Event / Policy / Decision / Approval / Agent
  - **8 Edge Type 枚举锁定**：BELONGS_TO / TRADES_WITH / PRODUCES / USES_ASSET / EVIDENCED_BY / GOVERNED_BY / APPROVED_BY / DERIVED_FROM
  - **Node Contract（12 字段）**：nodeId / nodeType / entityId / tenantId / version / status / createdAt / updatedAt / source / sourceEvidenceId / evidenceRefs / properties
  - **Edge Contract（11 字段）**：edgeId / edgeType / fromEntity / toEntity / tenantId / validity / version / sourceEvent / sourceEvidenceId / createdAt / properties
  - **关键原则**：边必由 Domain Event 驱动创建，禁止 Graph AI/heuristic 写边（lint 校验）
  - **每边必有 sourceEvent + sourceEvidenceId**（可追溯至 Outbox Event → Domain Mutation → Evidence）
  - 实现 Projection Rule 引擎（Domain Event → Cypher MERGE）
  - 实现 Contract 编译期/CI 校验
- **验收标准**：
  - [ ] 24 Node Type 全部定义且无遗漏（Approval 为一等公民）
  - [ ] 8 Edge Type 全部定义且无遗漏
  - [ ] Node Contract 12 字段 + Edge Contract 11 字段全部落地
  - [ ] 边必由 Domain Event 驱动（lint 校验通过，禁止 Graph AI 写边）
  - [ ] 每边必有 sourceEvent + sourceEvidenceId（追溯链完整）
  - [ ] Projection Rule 示例（order.created.v1 → Cypher MERGE）执行成功
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-014-graph-contract-evidence.json`（含 24 Node + 8 Edge 定义、Contract 校验报告、Projection Rule 执行日志、lint 报告）
- **依赖任务**：EBCX-EV1-005
- **遵守红线**：TASK-R06（Neo4j Projection，边由 Domain Event 驱动）、TASK-R09（Evidence First）

### EBCX-EV1-015：实现 Observability 基础（日志/指标/链路追踪 + Evidence 关联）
- **关联 spec.md**：§4.4（可维护性全模块）、§5.9.1 规则 4（可观测性）
- **关联 design.md**：D22 Observability、D-GATE-05（6 项关键指标）
- **实现内容**：
  - 接入华为云 APM/云监控/日志服务
  - 实现结构化 JSON 日志（含 traceId / spanId / tenantId / operator / businessObjectId / evidenceId / action / message）
  - 实现 Evidence 日志独立通道（append-only，与普通业务日志隔离）
  - 接入 OpenTelemetry 全链路追踪（REST 入口到 Evidence 写入串联单一 traceId）
  - 实现关键指标采集：核心交易 QPS / 延迟分位（P50/P95/P99）/ 错误率 / Evidence 写入速率 / Graph 投影延迟 / Agent 执行成功率 / Policy 命中率 / RLS 拦截次数
  - 实现 D-GATE-05 6 项指标：event_lag / projection_lag / consumer_lag / rebuild_duration / failed_projection_count / dlq_count
  - 实现告警：核心交易错误率 >0.1% / Graph 投影延迟 >3s / RLS 拦截异常 / Agent 治理拒绝异常
- **验收标准**：
  - [ ] 结构化 JSON 日志含全部必需字段
  - [ ] Evidence 日志独立 append-only 通道验证
  - [ ] 全链路 traceId 串联（OpenTelemetry）
  - [ ] 关键指标 + 6 项 D-GATE-05 指标全部接入监控
  - [ ] 告警触发验证（模拟错误率 >0.1% 触发告警）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-015-observability-evidence.json`（含日志样本、trace 链路截图、指标监控面板、告警测试）
- **依赖任务**：EBCX-EV1-005
- **遵守红线**：TASK-R09（Evidence First）

### EBCX-EV1-016：实现 Benchmark Harness 框架（B1 Profile + Load Generator）🔴 关键
- **关联 spec.md**：§4.1 规则 3（Benchmark Profile B1~B5）
- **关联 design.md**：D21 Benchmark Architecture、D-GATE-04（B1-EBCX-BASELINE Profile 绑定）
- **实现内容**：
  - 建立 B1-EBCX-BASELINE Profile 框架（Hardware + Workload + Target 绑定）
  - **Hardware 锁定**：16 vCPU / 64 GB RAM / PostgreSQL 8vCPU 32GB NVMe 2TB / Kafka 3 broker 4vCPU 16GB / Redis 4vCPU 16GB / OpenSearch 3 node 4vCPU 16GB / Neo4j 4vCPU 16GB
  - **Workload 锁定**：50 租户 / 10000 用户 / 1000 万 Order / 500 万 Contract / 500 万 Invoice / 200 万 Payment / 2000 并发 / Transaction Mix Order 40% Contract 20% Invoice 20% Payment 20% / 同步复制开启 / Evidence Write Ratio 100% / Graph Projection 异步 Normal Mode
  - **Target 锁定**：TPS ≥2000 / P95 ≤500ms / Error ≤0.1%
  - 实现 Load Generator（Locust 或 k6）
  - 实现 Dataset 生成脚本（1000 万 Order 等）
  - 实现 Scenario 定义（Create Order + validation + tenant RLS + commit + evidence append + audit）
  - 实现 Measurement + Report（P50/P95/P99/TPS/ErrorRate/Hardware/Workload/ConsistencyMode）
  - 实现 Measured Baseline 路径：`{repo}/benchmarks/baselines/b1/{date}.json`，版本化管理
- **验收标准**：
  - [ ] B1-EBCX-BASELINE Profile 框架可执行
  - [ ] Load Generator 可模拟 2000 并发
  - [ ] Dataset 生成脚本可生成 1000 万 Order
  - [ ] Measurement + Report 产出完整 JSON 报告
  - [ ] Measured Baseline 版本化纳入 git
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-016-benchmark-harness-evidence.json`（含 B1 Profile 定义、Load Generator 配置、Dataset 生成日志、框架自检报告）
- **依赖任务**：EBCX-EV1-001
- **遵守红线**：TASK-R08（B1 性能指标必须有 Benchmark Harness）

## 1.4 API 与事件契约

### EBCX-EV1-017：实现 REST API /api/v1/rel/* 第一入口
- **关联 spec.md**：§5.3.1 规则 4（REST 第一入口）、§4.5 规则 1（版本兼容）、§5.7.1 规则 3（统一认证）
- **关联 design.md**：D16 API / GraphQL（REST 第一入口 + 接口分类 + 统一 Header + 统一错误码）
- **实现内容**：
  - 实现 REST API 第一入口 `/api/v1/rel/*`
  - 实现接口分类：Command（Mutation）`/api/v1/rel/{module}/commands/*` / Query（Read）`/api/v1/rel/{module}/queries/*` / Evidence / Policy / Agent / Graph
  - 实现统一 Header：`Authorization: Bearer <JWT>` / `X-Tenant-Id` / `X-Country-Pack` / `X-Trace-Id`
  - 实现统一错误码：`EBCX-{MODULE}-{REASON}`（如 EBCX-EVIDENCE-WRITE-FAIL）
  - 实现版本化路径管理（破坏性变更新增版本号，旧版本保留 ≥2 个 EV 周期）
  - 复用 HTKIS-AF OAuth2 + JWT 认证
- **验收标准**：
  - [ ] REST API 第一入口可访问
  - [ ] 接口分类全部实现
  - [ ] 统一 Header + 错误码校验通过
  - [ ] 版本兼容策略验证（v1 保留，v2 发布）
  - [ ] 未携带 JWT 返回 401
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-017-rest-api-evidence.json`（含 API 文档、测试报告、版本兼容验证）
- **依赖任务**：EBCX-EV1-007
- **遵守红线**：TASK-R05（Read/Query 不强制走 Mutation Pipeline）、TASK-R09（Evidence First）

### EBCX-EV1-018：实现 GraphQL 适配层（复用 REST 认证）
- **关联 spec.md**：§5.3.1 规则 4（GraphQL 适配层）、§5.7.1 规则 3（统一认证）
- **关联 design.md**：D16 API / GraphQL（GraphQL 适配层契约）
- **实现内容**：
  - 实现 GraphQL 适配层 `/api/v1/graphql`
  - 复用 REST 认证体系（OAuth2 + JWT）
  - Query 类型映射 REST Query 接口，Mutation 类型映射 REST Command 接口
  - 嵌套查询深度限制 ≤3
  - 图查询优先走 `/api/v1/rel/graph/queries/*` REST 接口
- **验收标准**：
  - [ ] GraphQL 适配层可访问
  - [ ] 复用 REST 认证（同一 JWT）
  - [ ] 嵌套深度 >3 被拒绝
  - [ ] 图查询走 REST 接口
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-018-graphql-evidence.json`
- **依赖任务**：EBCX-EV1-017
- **遵守红线**：TASK-R05、TASK-R09

### EBCX-EV1-019：实现 Event Contract + Schema Registry
- **关联 spec.md**：§5.4（事件模型）、§4.5 规则 1（版本兼容）
- **关联 design.md**：D17 Event Contract（事件命名 + schema + 版本化 + 向后兼容）
- **实现内容**：
  - 实现事件命名规范：`{module}.{aggregate}.{action}.v{n}`
  - 实现统一 schema 字段：event_id / event_type / event_version / tenant_id / aggregate_id / aggregate_version / trace_id / span_id / occurred_at / payload / evidence_ref / source_event_ref
  - 集成华为云 DMS Schema Registry（或等价）
  - 实现向后兼容策略：新增字段可选 / 删除重命名需新增版本 / payload schema 注册至 Registry
  - 实现消费者按版本订阅
- **验收标准**：
  - [ ] 事件命名规范校验通过
  - [ ] 统一 schema 字段全部落地
  - [ ] Schema Registry 注册成功
  - [ ] 向后兼容策略验证（新增字段兼容，删除字段需新版本）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-019-event-contract-evidence.json`
- **依赖任务**：EBCX-EV1-004
- **遵守红线**：TASK-R05、TASK-R09

### EBCX-EV1-020：实现 CQRS Command/Query 分离基础
- **关联 spec.md**：§5.1.1 规则 1（Read/Query 分流）、§5.4.1 规则 6（Event-Native + CQRS + Evidence Ledger）
- **关联 design.md**：D08 CQRS（Command/Query 分离 + Read Model 投影）
- **实现内容**：
  - 实现 Command 侧：CommandBus → Aggregate → Transaction → Event → Evidence → Outbox → Projection
  - 实现 Query 侧：简单查询 PostgreSQL 直查（受 RLS） / 图查询 Neo4j / 聚合查询物化视图 / 溯源查询三层联合
  - 实现物化视图策略（高频聚合查询由 Outbox Event 触发增量刷新）
  - **Query 路径不触发 Agent 环节**（避免治理过度）
- **验收标准**：
  - [ ] Command/Query 分离校验通过
  - [ ] Query 路径不触发 Agent（lint 通过）
  - [ ] 物化视图刷新 ≤3s
  - [ ] 查询 P95 满足 §4.1 规则 2（单跳/两跳 ≤200ms，三跳+ ≤1s）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-020-cqrs-evidence.json`
- **依赖任务**：EBCX-EV1-005、EBCX-EV1-017
- **遵守红线**：TASK-R05（Read/Query 不强制走 Mutation Pipeline）、TASK-R09

## 1.5 DevSecOps 与 IaC

### EBCX-EV1-021：实现 DevSecOps CI/CD 流水线
- **关联 spec.md**：§5.9.1 规则 2（DevSecOps）、§5.12.1 红线六
- **关联 design.md**：D24 Deployment Architecture（DevSecOps CI/CD 流水线）
- **实现内容**：
  - 实现 CI/CD 流水线：构建（Go + Rust 双语言）→ SAST（静态分析）→ DAST（动态分析）→ 依赖扫描 → 镜像扫描 → 自动化测试（Go 459 + Rust 297）→ 部署 staging → 经评审部署 prod（多 AZ）
  - 集成四类安全扫描工具
  - 实现流水线失败阻断（任一扫描未通过阻断部署，EBCX-DEVSECOPS-SCAN-FAIL）
  - 实现环境分层：dev / staging / prod
- **验收标准**：
  - [ ] CI/CD 流水线可执行
  - [ ] 四类安全扫描全部集成
  - [ ] 扫描失败阻断部署验证
  - [ ] 环境分层部署验证
  - [ ] Go 459 + Rust 297 测试全部通过
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-021-devsecops-evidence.json`（含流水线配置、扫描报告、测试报告）
- **依赖任务**：EBCX-EV1-001
- **遵守红线**：TASK-R09（Evidence First）

### EBCX-EV1-022：实现 IaC 基础设施（Terraform + 华为云 CCE/RDS/OBS/DMS）
- **关联 spec.md**：§5.9.1 规则 1/3/5（云原生 + IaC + 多 AZ）、§5.12.1 红线六
- **关联 design.md**：D24 Deployment Architecture（IaC 管理 + 多 AZ 高可用）
- **实现内容**：
  - 使用 Terraform 管理华为云资源（CCE / RDS PostgreSQL / OBS / DMS Kafka / Neo4j / APM）
  - 实现多 AZ 高可用部署（PostgreSQL 主备同步 + 跨 AZ）
  - 实现禁止手工变更生产基础设施（IaC 提交 + 评审）
  - 保持可迁移性不产生不可逆厂商锁定
- **验收标准**：
  - [ ] Terraform 可一键部署全部华为云资源
  - [ ] 多 AZ 高可用验证（单 AZ 故障服务不中断）
  - [ ] 手工变更被拒绝并审计
  - [ ] 可迁移性验证（核心能力可在标准云原生环境运行）
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-022-iac-evidence.json`（含 Terraform 配置、部署日志、多 AZ 故障切换测试）
- **依赖任务**：EBCX-EV1-021
- **遵守红线**：TASK-R09（Evidence First）

## 1.6 B1 Benchmark 执行与 Gate 评审

### EBCX-EV1-023：执行 B1 Benchmark 首次压测 + 产出 Measured Baseline 🔴 关键
- **关联 spec.md**：§4.1 规则 3（B1 Profile）
- **关联 design.md**：D21 Benchmark Architecture、D-GATE-04（B1-EBCX-BASELINE Profile 绑定）
- **实现内容**：
  - 部署 B1-EBCX-BASELINE Hardware（16 vCPU / 64 GB RAM 等）
  - 生成 B1 Workload Dataset（50 租户 / 10000 用户 / 1000 万 Order 等）
  - 执行 B1 Scenario（Create Order + validation + tenant RLS + commit + evidence append + audit）
  - 测量 P50/P95/P99/TPS/ErrorRate
  - 产出 Measured Baseline 至 `{repo}/benchmarks/baselines/b1/{date}.json`
  - 验证 B1 Target：TPS ≥2000 / P95 ≤500ms / Error ≤0.1%
  - 若未达成，分析瓶颈并优化（不允许降低 B1 Target，必须绑定 B1-EBCX-BASELINE Profile）
- **验收标准**：
  - [ ] B1 Hardware + Workload 部署完成
  - [ ] B1 Scenario 执行成功
  - [ ] Measured Baseline 产出并版本化
  - [ ] B1 Target 达成：TPS ≥2000 / P95 ≤500ms / Error ≤0.1%
  - [ ] 若未达成，瓶颈分析报告 + 优化措施
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-023-b1-baseline-evidence.json`（含 B1 压测报告、Measured Baseline JSON、Target 达成验证、瓶颈分析（若有））
- **依赖任务**：EBCX-EV1-016、EBCX-EV1-022、EBCX-EV1-009、EBCX-EV1-012
- **遵守红线**：TASK-R08（B1 必须有 Benchmark Harness）、TASK-R09（Evidence First）、TASK-R10（Digital Engineering 闭环）

### EBCX-EV1-024：EV1 Gate 评审准备（交付物汇总 + 一致性校验）
- **关联 spec.md**：§5.10 EV1 Gate、§5.12.1 红线七
- **关联 design.md**：D01~D24 一致性映射
- **实现内容**：
  - 汇总 EV1 全部交付物（24 个任务的 Evidence 产出）
  - 校验 EV1 与 spec.md v1.1 + design.md v1.1 一致性
  - 校验 10 条施工红线遵守情况
  - 校验 8 条架构红线 + 7 条母架构约束对齐
  - 生成 EV1 Gate 评审报告（进入条件 / 交付物 / 验收标准三要素）
- **验收标准**：
  - [ ] 24 个任务全部完成且 Evidence 产出齐全
  - [ ] 与 spec.md v1.1 + design.md v1.1 一致性校验通过
  - [ ] 10 条施工红线 + 8 条架构红线 + 7 条母架构约束全部遵守
  - [ ] EV1 Gate 评审报告三要素齐全
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-024-gate-review-evidence.json`（含交付物清单、一致性校验报告、红线遵守报告、Gate 评审报告）
- **依赖任务**：EBCX-EV1-001~023 全部
- **遵守红线**：TASK-R01~R10 全部

---

# 二、EV2 Transaction Core 细化任务

> **目标**：建立 Transaction Core（Order/Contract/Invoice/Payment）+ Orchestrator 11 阶段编排 + Policy Engine + Agent Runtime 基础，实现第一性原理链路闭环。
> **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路、§5.4 事件模型、§5.5 Evidence Graph、§5.6 Governed Agent、§5.13 Data Lineage
> **关联 design.md**：D04 Transaction Core、D09 Data Lineage、D10 Policy Engine、D11 Agent Runtime、D-GATE-02（Saga/Workflow 边界）、D-GATE-07（Agent Authorization）

## 2.1 Transaction Core 聚合根

### EBCX-EV2-001：实现 Order 聚合根
- **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路、§6.1（数据约束）
- **关联 design.md**：D04 Transaction Core（OrderAggregate，金额一致性 + 状态机单调）
- **实现内容**：
  - 实现 `OrderAggregate`（含 orderId / customerId / items / status / version / sourceEvidenceId / totalAmount）
  - 实现状态机：Draft → Submitted → Approved → Executing → Verified → Completed / Failed → Compensating → Draft
  - 实现不变式：金额一致性（totalAmount = sum(items.amount)）、状态机单调推进
  - 实现命令处理：CreateOrder / SubmitOrder / ApproveOrder / ExecuteOrder / VerifyOrder / CompleteOrder / CompensateOrder
  - 实现事件：OrderCreated / OrderSubmitted / OrderApproved / OrderExecuted / OrderVerified / OrderCompleted / OrderFailed / OrderCompensated
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Order 节点 + TRADES_WITH 边至 Customer）
- **验收标准**：
  - [ ] 状态机单调推进校验通过（非法状态转换被拒绝）
  - [ ] 金额一致性不变式校验通过
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Event Replay 可重建 Order 状态
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-001-order-evidence.json`
- **依赖任务**：EBCX-EV1-012（MasterData Customer）、EBCX-EV1-014（Graph Contract）
- **遵守红线**：TASK-R03（聚合根边界）、TASK-R04（Evidence First）、TASK-R05（Mutation 进入治理链）、TASK-R09

### EBCX-EV2-002：实现 Contract 聚合根
- **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路
- **关联 design.md**：D04 Transaction Core（ContractAggregate）
- **实现内容**：
  - 实现 `ContractAggregate`（含 contractId / orderId / terms / status / version / sourceEvidenceId）
  - 实现状态机 + 不变式
  - 实现命令处理 + 事件
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Contract 节点 + BELONGS_TO 边至 Order）
- **验收标准**：
  - [ ] 状态机 + 不变式校验通过
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-002-contract-evidence.json`
- **依赖任务**：EBCX-EV2-001
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R09

### EBCX-EV2-003：实现 Invoice 聚合根
- **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路
- **关联 design.md**：D04 Transaction Core（InvoiceAggregate）
- **实现内容**：
  - 实现 `InvoiceAggregate`（含 invoiceId / orderId / contractId / paymentId / amount / version / sourceEvidenceId）
  - 实现状态机 + 不变式
  - 实现命令处理 + 事件
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Invoice 节点 + BELONGS_TO 边至 Order/Contract）
- **验收标准**：
  - [ ] 状态机 + 不变式校验通过
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-003-invoice-evidence.json`
- **依赖任务**：EBCX-EV2-002
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R09

### EBCX-EV2-004：实现 Payment 聚合根
- **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路
- **关联 design.md**：D04 Transaction Core（PaymentAggregate）
- **实现内容**：
  - 实现 `PaymentAggregate`（含 paymentId / invoiceId / amount / status / version / sourceEvidenceId）
  - 实现状态机 + 不变式
  - 实现命令处理 + 事件
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph（Payment 节点 + BELONGS_TO 边至 Invoice）
- **验收标准**：
  - [ ] 状态机 + 不变式校验通过
  - [ ] Evidence + Outbox + Graph 三层写入一致
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-004-payment-evidence.json`
- **依赖任务**：EBCX-EV2-003
- **遵守红线**：TASK-R03、TASK-R04、TASK-R05、TASK-R09

## 2.2 Orchestrator 与 Saga

### EBCX-EV2-005：实现 Transaction Orchestrator 11 阶段编排（D-GATE-02）🔴 关键
- **关联 spec.md**：§5.1.1（第一性原理链路 11 阶段）、§5.1.2（交互流程）
- **关联 design.md**：D04 Transaction Core、D-GATE-02（Transaction Orchestrator 与 Saga/Workflow 边界）
- **实现内容**：
  - 实现 `TransactionOrchestrator` 编排 11 阶段链路（Mutation 路径）：
    1. Business（加载主数据/权限上下文，同步读）
    2. Event（接收业务事件，同步）
    3. Transaction（聚合根命令，**Local ACID**）
    4. Data（持久化，**Local ACID**）
    5. Evidence（append-only，**Local ACID**）
    6. Policy 匹配（同步读 Graph）
    7. Decision（允许/拒绝/转审批，同步）
    8. Agent 执行（受治理，**Saga/Workflow**）
    9. Execution（业务动作，**Saga/Workflow**）
    10. Verification（独立验证，同步/异步）
    11. Closure（写入新可信 Evidence，**Local ACID**）
  - 实现 Read/Query 路径分流（不强制走完整 11 阶段链路）
  - **关键约束**：Orchestrator 是编排器，不是分布式事务管理器；11 阶段不是单一 ACID 事务
  - 实现 Local ACID + Domain Event + Saga/Workflow + Compensation + Evidence 五层组合
- **验收标准**：
  - [ ] 11 阶段链路全部实现且顺序不可变
  - [ ] Read/Query 路径不触发 Agent 环节
  - [ ] Local ACID 事务 P95 ≤500ms
  - [ ] 每步骤产生 Evidence
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-005-orchestrator-evidence.json`
- **依赖任务**：EBCX-EV2-001~004、EBCX-EV1-020
- **遵守红线**：TASK-R03（聚合根边界）、TASK-R05（Mutation 走完整链路）、TASK-R09、TASK-R10

### EBCX-EV2-006：实现 Saga/Workflow 跨服务编排 + Compensation
- **关联 spec.md**：§5.1.1（第一性原理链路）、§4.2 规则 4
- **关联 design.md**：D-GATE-02（Saga 补偿动作每步骤明确）
- **实现内容**：
  - 实现 Saga 编排：Order Created → Contract Approval → Invoice Issued → Payment Requested → Payment Verification → Closure
  - 实现每步骤补偿动作：
    | Saga 步骤 | 正向动作 | 补偿动作 |
    | Contract Approval | 发起审批 | 撤回审批请求 |
    | Invoice Issued | 开具发票 | 作废发票（红冲） |
    | Payment Requested | 请求外部支付 | 取消支付请求 / 退款 |
    | Payment Verification | 验证支付结果 | 标记支付未决，转人工 |
    | Closure | 写入新 Evidence | 写入补偿 Evidence（append-only，不删除原 Evidence） |
  - 实现步骤失败触发补偿回滚
  - 实现每步骤留证（含补偿动作也产生补偿 Evidence）
- **验收标准**：
  - [ ] Saga 编排 5 步骤全部实现
  - [ ] 每步骤补偿动作明确且可执行
  - [ ] 步骤失败触发补偿回滚验证
  - [ ] 补偿动作产生补偿 Evidence（append-only，不删除原 Evidence）
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-006-saga-evidence.json`
- **依赖任务**：EBCX-EV2-005
- **遵守红线**：TASK-R03、TASK-R04（补偿也产生 Evidence）、TASK-R09

### EBCX-EV2-007：实现 Domain Event 契约 + Schema Registry（Transaction 模块）
- **关联 spec.md**：§5.4（事件模型）、§4.5 规则 1
- **关联 design.md**：D17 Event Contract
- **实现内容**：
  - 实现 Transaction 模块 Domain Event 契约（OrderCreated / ContractSigned / InvoiceIssued / PaymentExecuted 等）
  - 注册至 Schema Registry
  - 实现向后兼容策略
- **验收标准**：
  - [ ] Domain Event 契约全部定义并注册
  - [ ] 向后兼容策略验证
  - [ ] Unit Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-007-domain-event-evidence.json`
- **依赖任务**：EBCX-EV1-019、EBCX-EV2-001~004
- **遵守红线**：TASK-R05、TASK-R09

### EBCX-EV2-008：实现 CQRS Read Model 投影（Transaction 模块）
- **关联 spec.md**：§5.1.1 规则 1（Read/Query 分流）、§5.4.1 规则 6
- **关联 design.md**：D08 CQRS
- **实现内容**：
  - 实现 Transaction 模块 Read Model 投影（OrderReadModel / ContractReadModel / InvoiceReadModel / PaymentReadModel）
  - 实现物化视图（高频聚合查询由 Outbox Event 触发增量刷新）
  - 实现溯源查询（Evidence Ledger + Graph + Artifact 三层联合）
- **验收标准**：
  - [ ] Read Model 投影正确
  - [ ] 物化视图刷新 ≤3s
  - [ ] 溯源查询返回完整引用链
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-008-read-model-evidence.json`
- **依赖任务**：EBCX-EV1-020、EBCX-EV2-007
- **遵守红线**：TASK-R05（Read/Query 不走 Mutation Pipeline）、TASK-R09

### EBCX-EV2-009：实现 Evidence Provenance & Data Lineage
- **关联 spec.md**：§5.13（Data Lineage 全模块）、§5.4（事件模型）
- **关联 design.md**：D09 Data Lineage
- **实现内容**：
  - 实现完整 Data Lineage 链路：Data→Source→Event→Transaction→Evidence→Transformation→Decision→Agent→Execution→Verification
  - 实现 Provenance 记录（每条 Evidence 含 sourceEventId / transactionId / operatorId / policyId / verificationId 引用）
  - 实现溯源子图查询接口：`GET /api/v1/rel/evidence/lineage/{businessObjectId}`
  - 实现断链处理：Data Lineage 缺失环节时标记"Lineage 不完整"，告警并阻断作为决策依据（EBCX-LINEAGE-BROKEN）
  - 实现业务对象溯源子图（Invoice/Order/Contract/Payment）
- **验收标准**：
  - [ ] Data Lineage 链路完整可追溯
  - [ ] Provenance 引用完整性校验通过
  - [ ] 溯源子图查询 P95 ≤1s
  - [ ] 断链阻断决策验证
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-009-lineage-evidence.json`
- **依赖任务**：EBCX-EV2-008
- **遵守红线**：TASK-R09（Evidence First）、TASK-R10（Digital Engineering 闭环）

## 2.3 Policy Engine 与 Agent Runtime

### EBCX-EV2-010：实现 Policy Engine 基础
- **关联 spec.md**：§5.5 治理关系、§5.6 Agent 治理、§6.4 Policy 数据约束
- **关联 design.md**：D10 Policy Engine
- **实现内容**：
  - 实现 `PolicyRule` 聚合根（含 policyId / policyType / scope / condition / action / version / effectiveFrom / effectiveTo）
  - 实现 policyType：权限 / 审批 / 计算 / 禁止
  - 实现 scope：节点类型 / 边类型 / 租户 / 组织
  - 实现 condition：可执行表达式或规则集（基于 Evidence Graph 子图匹配）
  - 实现 action：允许 / 拒绝 / 转人工审批
  - 实现评估流程：Orchestrator → Policy Engine → Neo4j 查询子图 → 规则匹配 → Decision
  - 实现与 Approval 交互（裁决为"转审批"时发起 Approval 流转）
  - 实现版本单调 + 生效区间不重叠
- **验收标准**：
  - [ ] Policy 评估 ≤50ms
  - [ ] 规则版本单调 + 生效区间不重叠校验通过
  - [ ] 与 Approval 交互验证
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-010-policy-evidence.json`
- **依赖任务**：EBCX-EV1-014、EBCX-EV2-005
- **遵守红线**：TASK-R07（Agent Execution 经 Policy）、TASK-R09

### EBCX-EV2-011：实现 Approval 聚合根（Evidence Graph 一等公民）
- **关联 spec.md**：§5.5.1 规则 1（Approval 一等公民）、§5.5.1 规则 3（审批关系）
- **关联 design.md**：D06 Evidence Graph Projection（Approval 一等公民）、D-GATE-06
- **实现内容**：
  - 实现 `ApprovalAggregate`（含 approvalId / decisionId / approverId / status / sourceEvidenceId / version）
  - 实现 Approval 作为 Evidence Graph 一等公民节点（nodeType='Approval'）
  - 实现审批关系边：Decision→Approval→Execution、Approval→Evidence（APPROVED_BY 边）
  - 实现审批流转事实固化为 Evidence
  - 写入 Evidence Ledger + Outbox Event + 投影至 Neo4j Graph
- **验收标准**：
  - [ ] Approval 作为 Graph 一等公民节点存在
  - [ ] 审批关系边全部定义（APPROVED_BY）
  - [ ] 审批流转事实固化为 Evidence
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-011-approval-evidence.json`
- **依赖任务**：EBCX-EV2-010
- **遵守红线**：TASK-R06（边由 Domain Event 驱动）、TASK-R09

### EBCX-EV2-012：实现 Agent Runtime 基础（Governed Agent 三重治理）
- **关联 spec.md**：§5.6（Governed Agent 全模块）、§5.12.1 红线四/五
- **关联 design.md**：D11 Agent Runtime、§2.5.5 Governed Agent 三重治理
- **实现内容**：
  - 实现 `AgentRuntime` 组件
  - 实现三重治理流程（顺序不可变）：权限校验 → Policy 匹配 → 审批流转
  - 实现 Provenance 注入（Agent 决策上下文查询时 Evidence Graph 返回子图同时附带 Provenance 引用）
  - 实现 Agent 可读 Current State + Evidence + Policy + Master Data，但影响决策的关键事实必须可追溯 Evidence
  - 实现执行记录（§6.5）：agentExecutionId / agentId / inputEvidenceIds / policyDecisions / executionAction / verificationResult / outputEvidenceId / auditTrail / createdAt
  - 实现 Verification 闭环（执行后产生 Verification 结果，通过后写入新 Evidence，未通过触发补偿）
  - 实现同步路径 ≤2s 或异步返回任务 ID
- **验收标准**：
  - [ ] 三重治理顺序不可变验证
  - [ ] Provenance 注入验证
  - [ ] Agent 决策关键事实无 Evidence Provenance 被拒绝（EBCX-AGENT-POLICY-REJECT）
  - [ ] Verification 闭环验证
  - [ ] 同步路径 ≤2s 或异步返回任务 ID
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-012-agent-runtime-evidence.json`
- **依赖任务**：EBCX-EV2-010、EBCX-EV2-011
- **遵守红线**：TASK-R07（Agent Execution 经 Policy/Authorization）、TASK-R09

### EBCX-EV2-013：实现 Agent Execution Authorization 模型（D-GATE-07）🔴 关键
- **关联 spec.md**：§5.6（Governed Agent）、§5.6.1 规则 1/4/5、§5.12.1 红线四/五、§6.5
- **关联 design.md**：D11 Agent Runtime、D-GATE-07（Agent Execution Authorization 模型）
- **实现内容**：
  - 实现 Agent 执行链（不可变）：Agent → Reason → Policy → Approval → Authorize → Execute → Verify → Evidence
  - 实现 Authorization 模型必含要素：
    - Agent Identity（agentId + version，注册制）
    - Execution Scope（Policy 显式授权）
    - Tenant Scope（RLS 强制）
    - Policy Decision（Policy Engine 独立裁决）
    - Approval Requirement（Approval 一等公民）
    - **Execution Token**（Authorizer 签发，短期有效，绑定 agentId + scope + tenant + expiry）
    - Idempotency Key（agentExecutionId + idempotencyKey）
    - **Independent Verifier**（独立验证，非 Agent 自证）
    - Evidence（执行结果 Evidence，append-only）
  - 实现 Authorizer 组件（签发 Execution Token）
  - 实现 Independent Verifier 组件（与 Agent 解耦）
  - **禁止项**：Agent 自授 Execution Token / Agent 自证 / Agent 绕过 Policy / Agent 跨租户 / Agent 执行未授权能力
- **验收标准**：
  - [ ] Agent 不能自授 Execution Token（必须由 Authorizer 签发）
  - [ ] Independent Verifier 与 Agent 解耦
  - [ ] Agent 绕过 Policy 直接 Execute 被拒绝
  - [ ] Agent 跨租户执行被 RLS 拦截
  - [ ] Agent 执行未授权能力被拒绝
  - [ ] Agent 自证成功后写入 Evidence 被拒绝
  - [ ] Execution Token 短期有效 + 过期作废
  - [ ] 幂等键去重验证
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-013-agent-authz-evidence.json`
- **依赖任务**：EBCX-EV2-012
- **遵守红线**：TASK-R07（Agent Execution 必须经过 Policy/Authorization）、TASK-R09

## 2.4 API 与 Benchmark

### EBCX-EV2-014：实现 REST API Transaction/Evidence/Policy/Agent/Graph 模块
- **关联 spec.md**：§5.3.1 规则 4、§5.13（Data Lineage 查询）、§5.6（Agent 治理）
- **关联 design.md**：D16 API / GraphQL、D09 Data Lineage
- **实现内容**：
  - 实现 Transaction 模块 API：`POST /api/v1/rel/transaction/commands/create-order` 等
  - 实现 Evidence 模块 API：`GET /api/v1/rel/evidence/lineage/{businessObjectId}`
  - 实现 Policy 模块 API：`/api/v1/rel/policy/*`
  - 实现 Agent 模块 API：`POST /api/v1/rel/agent/commands/execute`
  - 实现 Graph 模块 API：`POST /api/v1/rel/graph/queries/subgraph`
  - 复用 HTKIS-AF 认证 + 统一 Header + 统一错误码
- **验收标准**：
  - [ ] 5 个模块 API 全部实现并可访问
  - [ ] 核心交易接口 P95 ≤500ms
  - [ ] Graph 单跳/两跳 P95 ≤200ms，三跳+ P95 ≤1s
  - [ ] Agent 同步路径 ≤2s 或异步返回任务 ID
  - [ ] Unit Test + Integration Test 通过
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-014-api-evidence.json`
- **依赖任务**：EBCX-EV1-017、EBCX-EV2-009、EBCX-EV2-013
- **遵守红线**：TASK-R05、TASK-R07、TASK-R09

### EBCX-EV2-015：实现 B2~B5 Profile 预留框架
- **关联 spec.md**：§4.1 规则 3（B1~B5 Profile）
- **关联 design.md**：D21 Benchmark Architecture、D-GATE-04（B2~B5 预留框架）
- **实现内容**：
  - 预留 B2-Standard-Enterprise Profile（8 vCPU / 32 GB / 10 租户 / 2000 用户 / 500 并发 / P95≤800ms / ≥500 TPS）
  - 预留 B3-Large-Enterprise Profile（32 vCPU / 128 GB / 200 租户 / 50000 用户 / 5000 并发 / P95≤1s / ≥5000 TPS）
  - 预留 B4-High-Concurrency Profile（64 vCPU / 256 GB / 500 租户 / 100000 用户 / 10000 并发 / P95≤2s / ≥10000 TPS）
  - 预留 B5-Extreme Profile（定制 / profile-defined）
  - 各 Profile 框架可执行但 Target 待后续 EV 细化
- **验收标准**：
  - [ ] B2~B5 Profile 框架全部预留
  - [ ] 各 Profile 可执行（Target 待细化）
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-015-b2-b5-evidence.json`
- **依赖任务**：EBCX-EV1-016
- **遵守红线**：TASK-R08

### EBCX-EV2-016：执行 B1 Benchmark（含 Transaction Core）+ Measured Baseline 🔴 关键
- **关联 spec.md**：§4.1 规则 3
- **关联 design.md**：D21 Benchmark Architecture、D-GATE-04
- **实现内容**：
  - 在 EV1 B1 基础上加入 Transaction Core 完整链路（Order + Contract + Invoice + Payment + Orchestrator + Policy + Agent + Evidence）
  - 执行 B1 Scenario（Transaction Mix: Order 40% / Contract 20% / Invoice 20% / Payment 20%）
  - 测量 P50/P95/P99/TPS/ErrorRate
  - 产出 Measured Baseline 至 `{repo}/benchmarks/baselines/b1/{date}.json`
  - 验证 B1 Target：TPS ≥2000 / P95 ≤500ms / Error ≤0.1%
  - 验证 Graph Projection Normal Mode P95 lag ≤3s
- **验收标准**：
  - [ ] B1 Scenario（含 Transaction Core）执行成功
  - [ ] Measured Baseline 产出并版本化
  - [ ] B1 Target 达成：TPS ≥2000 / P95 ≤500ms / Error ≤0.1%
  - [ ] Graph Projection P95 lag ≤3s（Normal Mode）
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-016-b1-baseline-evidence.json`
- **依赖任务**：EBCX-EV1-023、EBCX-EV2-014
- **遵守红线**：TASK-R08、TASK-R09、TASK-R10

### EBCX-EV2-017：EV2 Gate 评审准备
- **关联 spec.md**：§5.10 EV2 Gate、§5.12.1 红线七
- **关联 design.md**：D01~D24 一致性映射
- **实现内容**：
  - 汇总 EV2 全部交付物（17 个任务的 Evidence 产出）
  - 校验 EV2 与 spec.md v1.1 + design.md v1.1 一致性
  - 校验 10 条施工红线 + 8 条架构红线 + 7 条母架构约束
  - 生成 EV2 Gate 评审报告
- **验收标准**：
  - [ ] 17 个任务全部完成且 Evidence 产出齐全
  - [ ] 一致性校验通过
  - [ ] 红线全部遵守
  - [ ] EV2 Gate 评审报告三要素齐全
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-017-gate-review-evidence.json`
- **依赖任务**：EBCX-EV2-001~016 全部
- **遵守红线**：TASK-R01~R10 全部

---

# 三、EV3~EV12 里程碑级任务

> **说明**：EV3~EV12 为里程碑级任务，每个 EV 阶段含进入条件、交付物、验收标准三要素。具体子任务在各 EV 阶段启动时由 spec-task-agent 细化。
> **关联 spec.md**：§5.10 EV3~EV12 Gate 体系
> **关联 design.md**：D01~D24 对应模块

### EBCX-EV3-001：EV3 Evidence Core 里程碑（Evidence Graph 完整能力）
- **关联 spec.md**：§5.10 EV3 Evidence Core、§5.5 Evidence Graph
- **关联 design.md**：D06 Evidence Graph Projection、D09 Data Lineage
- **实现内容**：完善 Evidence Graph 完整能力（24 类节点 + 8 类边全量投影 + Data Lineage 完整链路 + 重建策略 + 模式分级监控）
- **验收标准**：[ ] Evidence Graph 24 类节点 + 8 类边全量投影；[ ] Data Lineage 完整链路；[ ] 重建 RTO ≤30min；[ ] 模式分级监控全部接入
- **Evidence 产出**：`evidence/ev3/EBCX-EV3-001-evidence-core-evidence.json`
- **依赖任务**：EBCX-EV2-017
- **遵守红线**：TASK-R01~R10

### EBCX-EV4-001：EV4 Finance 里程碑（财务核心）
- **关联 spec.md**：§5.10 EV4 Finance
- **关联 design.md**：D04 Transaction Core（Finance 模块）
- **实现内容**：财务核心模块（凭证 / 账簿 / 报表 / 合规），归并 EITP 财务能力，对接 Evidence 闭环
- **验收标准**：[ ] 财务核心模块可用；[ ] Evidence 闭环；[ ] 合规校验通过
- **Evidence 产出**：`evidence/ev4/EBCX-EV4-001-finance-evidence.json`
- **依赖任务**：EBCX-EV3-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV5-001：EV5 SCM 里程碑（供应链）
- **关联 spec.md**：§5.10 EV5 SCM
- **关联 design.md**：D04 Transaction Core（SCM 模块）
- **实现内容**：供应链模块（采购 / 库存 / 物流 / 供应商管理），对接 Evidence 闭环
- **验收标准**：[ ] SCM 模块可用；[ ] Evidence 闭环
- **Evidence 产出**：`evidence/ev5/EBCX-EV5-001-scm-evidence.json`
- **依赖任务**：EBCX-EV4-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV6-001：EV6 Manufacturing 里程碑（MES/生产）
- **关联 spec.md**：§5.10 EV6 Manufacturing、§5.2 规则 3（AirPLM/MES 归并）
- **关联 design.md**：D12 Industrial Core
- **实现内容**：制造核心模块（BOM / 工艺路线 / 工单 / 质检），归并 AirPLM/MES，对接 Evidence 闭环
- **验收标准**：[ ] 制造核心模块可用；[ ] AirPLM/MES 归并完成；[ ] Evidence 闭环
- **Evidence 产出**：`evidence/ev6/EBCX-EV6-001-manufacturing-evidence.json`
- **依赖任务**：EBCX-EV5-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV7-001：EV7 PLM/Quality/EAM 里程碑（工业核心）
- **关联 spec.md**：§5.10 EV7 PLM/Quality/EAM
- **关联 design.md**：D12 Industrial Core
- **实现内容**：PLM / Quality / EAM 模块（产品生命周期 / 质量管理 / 资产管理），对接 Evidence 闭环
- **验收标准**：[ ] PLM/Quality/EAM 模块可用；[ ] Evidence 闭环
- **Evidence 产出**：`evidence/ev7/EBCX-EV7-001-industrial-core-evidence.json`
- **依赖任务**：EBCX-EV6-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV8-001：EV8 Data Trust 里程碑（工业数据基础设施）
- **关联 spec.md**：§5.10 EV8 Data Trust、§5.11（工信部 1+4+N 政策对齐）
- **关联 design.md**：D15 Data Architecture
- **实现内容**：工业数据基础设施（数据资源库 / 数据技术库 / 工业数据标准库 / 高质量数据集库），对齐工信部 1+4+N
- **验收标准**：[ ] 4 库能力全部对齐；[ ] 政策对齐矩阵评审通过
- **Evidence 产出**：`evidence/ev8/EBCX-EV8-001-data-trust-evidence.json`
- **依赖任务**：EBCX-EV7-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV9-001：EV9 Agent Runtime 完整里程碑（企业智能体）
- **关联 spec.md**：§5.10 EV9 Agent Runtime、§5.6 Governed Agent
- **关联 design.md**：D11 Agent Runtime、D-GATE-07
- **实现内容**：Agent Runtime 完整能力（多类型 Agent / Agent 注册制 / Execution Token / Independent Verifier / Provenance 完整注入）
- **验收标准**：[ ] Agent Runtime 完整能力可用；[ ] 三重治理 + Authorization 模型全部落地
- **Evidence 产出**：`evidence/ev9/EBCX-EV9-001-agent-runtime-evidence.json`
- **依赖任务**：EBCX-EV8-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV10-001：EV10 Digital Twin 里程碑（企业数字孪生）
- **关联 spec.md**：§5.10 EV10 Digital Twin、§5.2 规则 3（SeaFusion-X 归并）
- **关联 design.md**：D12 Digital Twin Adapter
- **实现内容**：数字孪生完整能力（仿真 / 预测 / 验证），归并 SeaFusion-X，对接 Evidence Provenance
- **验收标准**：[ ] Digital Twin 完整能力可用；[ ] SeaFusion-X 归并完成；[ ] Verification 闭环
- **Evidence 产出**：`evidence/ev10/EBCX-EV10-001-digital-twin-evidence.json`
- **依赖任务**：EBCX-EV9-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV11-001：EV11 Industry Network 里程碑（产业链数据协同）
- **关联 spec.md**：§5.10 EV11 Industry Network
- **关联 design.md**：D14 Multi-Tenant Architecture
- **实现内容**：产业链数据协同（多企业 / 多租户 / 数据共享 / Evidence 互信）
- **验收标准**：[ ] 产业链协同能力可用；[ ] Evidence 互信建立
- **Evidence 产出**：`evidence/ev11/EBCX-EV11-001-industry-network-evidence.json`
- **依赖任务**：EBCX-EV10-001
- **遵守红线**：TASK-R01~R10

### EBCX-EV12-001：EV12 Global Platform 里程碑（全球化商业化）
- **关联 spec.md**：§5.10 EV12 Global Platform、§5.8 全球化架构
- **关联 design.md**：D14 Multi-Tenant Architecture、D20 RPO/RTO、D24 Deployment
- **实现内容**：全球化商业化（5 个 Country Pack 全量落地 / 多区域部署 / 灾备分级 / 合规校验）
- **验收标准**：[ ] 5 个 Country Pack 全量落地；[ ] 灾备分级按 §4.2 规则 3；[ ] 合规校验通过
- **Evidence 产出**：`evidence/ev12/EBCX-EV12-001-global-platform-evidence.json`
- **依赖任务**：EBCX-EV11-001
- **遵守红线**：TASK-R01~R10

---

# 四、集成测试与验证任务组

## 4.1 EV1 集成测试

### EBCX-EV1-IT-001：EV1 端到端集成测试（Enterprise Core 完整链路）
- **关联 spec.md**：§5.10 EV1、§5.1 第一性原理链路
- **关联 design.md**：D01~D24
- **实现内容**：
  - 端到端测试：创建 Enterprise → Organization → Person → MasterData → Permission，验证 Evidence + Outbox + Graph 三层一致
  - 验证 RLS 跨租户隔离
  - 验证 Evidence Ledger 四层纵深防御
  - 验证 Graph Projection 异步投影收敛 ≤3s
  - 验证 Observability 全链路 traceId 串联
- **验收标准**：
  - [ ] 端到端链路全部通过
  - [ ] 三层真相模型一致性验证
  - [ ] RLS 隔离验证
  - [ ] 四层防御验证
  - [ ] Graph 投影收敛验证
  - [ ] traceId 串联验证
- **Evidence 产出**：`evidence/ev1/EBCX-EV1-IT-001-e2e-evidence.json`
- **依赖任务**：EBCX-EV1-024
- **遵守红线**：TASK-R09、TASK-R10

## 4.2 EV2 集成测试

### EBCX-EV2-IT-001：EV2 端到端集成测试（Transaction Core 完整链路 + Agent 治理）
- **关联 spec.md**：§5.10 EV2、§5.1 第一性原理链路、§5.6 Governed Agent
- **关联 design.md**：D04、D11、D-GATE-02、D-GATE-07
- **实现内容**：
  - 端到端测试：Create Order → Submit → Policy 匹配 → Approval → Agent Execute → Verify → Closure，验证 11 阶段链路
  - 验证 Saga 跨服务编排 + Compensation
  - 验证 Agent 三重治理 + Authorization 模型（Execution Token + Independent Verifier）
  - 验证 Data Lineage 完整溯源子图
  - 验证 B1 Benchmark Target
- **验收标准**：
  - [ ] 11 阶段链路全部通过
  - [ ] Saga + Compensation 验证
  - [ ] Agent 治理 + Authorization 验证
  - [ ] Data Lineage 溯源验证
  - [ ] B1 Target 达成
- **Evidence 产出**：`evidence/ev2/EBCX-EV2-IT-001-e2e-evidence.json`
- **依赖任务**：EBCX-EV2-017
- **遵守红线**：TASK-R09、TASK-R10

## 4.3 回归测试

### EBCX-REG-001：存量功能回归测试（EITP/AirPLM/AeroForge/SeaFusion/HTKIS-AF/HyperDisk 归并）
- **关联 spec.md**：§4.5 规则 2（资产迁移）、§5.2 模块树
- **关联 design.md**：D15 Data Architecture（迁移策略）
- **实现内容**：
  - EITP 存量交易数据迁移至 Transaction Core，验证存量数据可读 + 新交易走 Evidence 闭环
  - AirPLM/MES 制造能力归并验证
  - AeroForge-X 证据治理归并验证
  - SeaFusion-X 数字孪生归并验证
  - HTKIS-AF 安全基座归并验证
  - HTKIZ/HyperDisk 存储归并验证
- **验收标准**：
  - [ ] 6 个既有项目归并全部验证
  - [ ] 存量数据可迁移 + 存量接口可适配
  - [ ] 新交易走 Evidence 闭环
- **Evidence 产出**：`evidence/reg/EBCX-REG-001-regression-evidence.json`
- **依赖任务**：EBCX-EV2-IT-001
- **遵守红线**：TASK-R01（不重新造轮子）、TASK-R09、TASK-R10

---

# 五、部署与配置任务组

### EBCX-DEPLOY-001：生产环境部署 + 多 AZ 高可用 + 灾备分级
- **关联 spec.md**：§5.9 华为云部署、§4.2 规则 3（RPO 分级）
- **关联 design.md**：D20 RPO/RTO、D24 Deployment、D-GATE-03（RPO≤0 技术实现条件）
- **实现内容**：
  - 部署生产环境至华为云 CCE（多 AZ 高可用）
  - PostgreSQL 主备同步 + quorum commit + 跨 AZ 同步复制（T0 RPO≤0）
  - 同城灾备（T1 RPO≤30s）+ 异地灾备（T2 RPO≤5min）
  - Neo4j 主备 + EventBus（DMS Kafka）3 broker 多副本
  - IaC 管理全部基础设施
  - DevSecOps CI/CD 流水线集成四类安全扫描
- **验收标准**：
  - [ ] 生产环境多 AZ 高可用部署完成
  - [ ] T0 RPO≤0 验证（同步复制 + quorum commit）
  - [ ] T1/T2 灾备分级验证
  - [ ] IaC 管理全部基础设施
  - [ ] DevSecOps 四类安全扫描集成
  - [ ] RTO ≤5min 验证
- **Evidence 产出**：`evidence/deploy/EBCX-DEPLOY-001-prod-deploy-evidence.json`
- **依赖任务**：EBCX-EV1-022、EBCX-EV2-017
- **遵守红线**：TASK-R08、TASK-R09、TASK-R10

### EBCX-DEPLOY-002：监控告警 + Evidence 关联配置
- **关联 spec.md**：§4.4 可维护性、§5.9.1 规则 4
- **关联 design.md**：D22 Observability、D-GATE-05（6 项指标）
- **实现内容**：
  - 配置华为云 APM/云监控/日志服务全量接入
  - 配置关键指标 + 6 项 D-GATE-05 指标告警阈值
  - 配置 Evidence 日志独立 append-only 通道
  - 配置 OpenTelemetry 全链路追踪
- **验收标准**：
  - [ ] 监控告警全部配置
  - [ ] 6 项 D-GATE-05 指标告警触发验证
  - [ ] Evidence 日志独立通道验证
  - [ ] traceId 串联验证
- **Evidence 产出**：`evidence/deploy/EBCX-DEPLOY-002-monitoring-evidence.json`
- **依赖任务**：EBCX-DEPLOY-001
- **遵守红线**：TASK-R09

---

# 六、评审与验证任务组

### EBCX-REVIEW-001：代码审查（关键代码 Review）
- **关联 spec.md**：§5.12 架构红线
- **关联 design.md**：D01~D24
- **实现内容**：
  - 审查 Evidence Ledger 四层纵深防御实现
  - 审查 Orchestrator 11 阶段编排 + Saga 边界
  - 审查 Agent Authorization 模型（Execution Token + Independent Verifier）
  - 审查 Graph Projection 边由 Domain Event 驱动
  - 审查 RLS + 多租户隔离
  - 审查 8 条架构红线 + 7 条母架构约束对齐
- **验收标准**：
  - [ ] 关键代码 Review 全部通过
  - [ ] 8 条架构红线 + 7 条母架构约束对齐
  - [ ] Review 意见全部 resolved
- **Evidence 产出**：`evidence/review/EBCX-REVIEW-001-code-review-evidence.json`
- **依赖任务**：EBCX-DEPLOY-002
- **遵守红线**：TASK-R01~R10

### EBCX-REVIEW-002：设计回顾（设计与实现一致性核对）
- **关联 spec.md**：spec.md v1.1 全文
- **关联 design.md**：design.md v1.1 全文
- **实现内容**：
  - 核对 spec.md v1.1 + design.md v1.1 与实现的一致性
  - 核对 D01~D24 全部 24 项设计决策落地
  - 核对 D-GATE-01~08 全部 8 项 Architecture Hardening 落地
  - 核对 13 Capability / 14 Module / 6 Bounded Context / 24 Entity 层级关系
- **验收标准**：
  - [ ] spec.md v1.1 + design.md v1.1 与实现一致性校验通过
  - [ ] D01~D24 全部落地
  - [ ] D-GATE-01~08 全部落地
  - [ ] 13/14/6/24 层级关系正确
- **Evidence 产出**：`evidence/review/EBCX-REVIEW-002-design-review-evidence.json`
- **依赖任务**：EBCX-REVIEW-001
- **遵守红线**：TASK-R01（不重新设计架构）、TASK-R10

### EBCX-REVIEW-003：变更确认（变更范围最终确认）
- **关联 spec.md**：§5.10 Gate 体系
- **关联 design.md**：D01~D24
- **实现内容**：
  - 确认 EV1 + EV2 变更范围
  - 确认 10 条施工红线 + 8 条架构红线 + 7 条母架构约束全部遵守
  - 确认 Physical Evidence 全部产出
  - 确认 Digital Engineering 闭环：Requirement → Design Decision → Task → Code → Test → Evidence → Verification → Closure
- **验收标准**：
  - [ ] 变更范围确认
  - [ ] 10 + 8 + 7 条红线全部遵守
  - [ ] Physical Evidence 全部产出
  - [ ] Digital Engineering 闭环验证
- **Evidence 产出**：`evidence/review/EBCX-REVIEW-003-change-review-evidence.json`
- **依赖任务**：EBCX-REVIEW-002
- **遵守红线**：TASK-R01~R10 全部

---

# 七、与 spec.md / design.md 一致性校验

## 7.1 spec.md 需求覆盖校验

| spec.md 章节 | 需求 | 覆盖任务 | 校验结果 |
|---|---|---|---|
| §4.1 性能 | B1 Benchmark Profile | EBCX-EV1-016/023、EBCX-EV2-016 | ✅ |
| §4.2 可靠性 | RPO 分级 + Evidence 不可篡改 + Neo4j 异步投影 | EBCX-EV1-003/004/005、EBCX-DEPLOY-001 | ✅ |
| §4.3 安全性 | OAuth2+JWT+RLS+审计+HTKIS-AF | EBCX-EV1-007 | ✅ |
| §4.4 可维护性 | 监控+日志+链路追踪 | EBCX-EV1-015、EBCX-DEPLOY-002 | ✅ |
| §4.5 兼容性 | 接口版本+资产迁移+全球化+云迁移 | EBCX-EV1-017/008、EBCX-REG-001 | ✅ |
| §5.1 第一性原理链路 | 11 阶段 + Read/Query 分流 | EBCX-EV2-005、EBCX-EV1-020 | ✅ |
| §5.2 模块树 | 三大核心 + 既有项目归并 | EBCX-EV1-001、EBCX-REG-001 | ✅ |
| §5.3 技术栈 | 云原生+PostgreSQL+Neo4j+OBS+REST+GraphQL+DDD+双语言+React | EBCX-EV1-001/002/005/006/017/018 | ✅ |
| §5.4 数据库与事件模型 | Event-Native+CQRS+Evidence Ledger+append-only | EBCX-EV1-003/020、EBCX-EV2-007/008 | ✅ |
| §5.5 Evidence Graph | 24 类节点+8 类边+Approval 一等公民 | EBCX-EV1-014、EBCX-EV2-011 | ✅ |
| §5.6 Governed Agent | 三重治理+Provenance+Verification | EBCX-EV2-012/013 | ✅ |
| §5.7 权限安全 | HTKIS-AF+RLS+统一认证+审计+租户隔离 | EBCX-EV1-007/008 | ✅ |
| §5.8 全球化 | Global Core+Country Pack+核心表不分叉 | EBCX-EV1-008、EBCX-EV12-001 | ✅ |
| §5.9 华为云部署 | CCE+DevSecOps+IaC+多 AZ | EBCX-EV1-021/022、EBCX-DEPLOY-001 | ✅ |
| §5.10 EV1~EV12 Gate | Gate 三要素 | EBCX-EV1-024、EBCX-EV2-017、EV3~EV12 | ✅ |
| §5.11 工信部 1+4+N | 政策对齐矩阵 | EBCX-EV8-001 | ✅ |
| §5.12 架构红线 | 8 条红线 | EBCX-REVIEW-001 | ✅ |
| §5.13 Data Lineage | 完整溯源子图+Provenance | EBCX-EV2-009 | ✅ |

**spec.md 覆盖结论**：§4.1~§5.13 全部 18 个章节需求均被任务覆盖，无遗漏。

## 7.2 design.md 设计决策覆盖校验

| design.md 设计项 | 主题 | 覆盖任务 | 校验结果 |
|---|---|---|---|
| D01 Logical Architecture | Modular Monolith + 14 模块 | EBCX-EV1-001 | ✅ |
| D02 Domain Boundary | 6 限界上下文 + 聚合根 | EBCX-EV1-009~013、EBCX-EV2-001~004 | ✅ |
| D03 Enterprise Core | 5 聚合根 | EBCX-EV1-009~013 | ✅ |
| D04 Transaction Core | Order/Contract/Invoice/Payment + Orchestrator | EBCX-EV2-001~005 | ✅ |
| D05 Evidence Ledger | append-only + 四层防御 | EBCX-EV1-003 | ✅ |
| D06 Evidence Graph Projection | 24 类节点 + 8 类边 + 异步投影 | EBCX-EV1-005/014 | ✅ |
| D07 Outbox + EventBus | 同事务原子 + 至少一次 + 幂等 | EBCX-EV1-004 | ✅ |
| D08 CQRS | Command/Query 分离 + Read Model | EBCX-EV1-020、EBCX-EV2-008 | ✅ |
| D09 Data Lineage | Provenance + 溯源子图 | EBCX-EV2-009 | ✅ |
| D10 Policy Engine | 规则模型 + 评估流程 | EBCX-EV2-010 | ✅ |
| D11 Agent Runtime | 三重治理 + Provenance 注入 | EBCX-EV2-012/013 | ✅ |
| D12 Digital Twin Adapter | SeaFusion-X 归并 + 验证层 | EBCX-EV10-001 | ✅ |
| D13 Security Architecture | HTKIS-AF + OAuth2+JWT+RLS+审计 | EBCX-EV1-007 | ✅ |
| D14 Multi-Tenant Architecture | RLS + Pack 路由 + 核心表不分叉 | EBCX-EV1-008 | ✅ |
| D15 Data Architecture | schema 划分 + 迁移 + OBS 命名 | EBCX-EV1-002/006 | ✅ |
| D16 API / GraphQL | REST 第一入口 + GraphQL 适配 | EBCX-EV1-017/018、EBCX-EV2-014 | ✅ |
| D17 Event Contract | 事件命名 + schema + 版本化 | EBCX-EV1-019、EBCX-EV2-007 | ✅ |
| D18 Consistency Model | 事务边界 + Outbox 原子 + Graph 最终一致 + Saga | EBCX-EV1-004、EBCX-EV2-005/006 | ✅ |
| D19 Failure / Recovery Model | Neo4j 故障不拖垮 + 重投 + 重建 | EBCX-EV1-005 | ✅ |
| D20 RPO / RTO | RPO 分级 + 灾备 + Country Pack | EBCX-DEPLOY-001、EBCX-EV12-001 | ✅ |
| D21 Benchmark Architecture | B1~B5 Profile + Measured Baseline | EBCX-EV1-016/023、EBCX-EV2-015/016 | ✅ |
| D22 Observability | 日志+指标+链路追踪+Evidence 关联 | EBCX-EV1-015、EBCX-DEPLOY-002 | ✅ |
| D23 Audit / Compliance | append-only 审计 + 不可篡改 + 数据出境 | EBCX-EV1-003/007 | ✅ |
| D24 Deployment Architecture | CCE + DevSecOps + IaC + 多 AZ | EBCX-EV1-021/022、EBCX-DEPLOY-001 | ✅ |

## 7.3 D-GATE-01~08 Architecture Hardening 覆盖校验

| D-GATE | 主题 | 覆盖任务 | 校验结果 |
|---|---|---|---|
| D-GATE-01 | Evidence Ledger 不可篡改安全边界 | EBCX-EV1-003 | ✅ 四层纵深防御 + Runtime/Migration Role 分离 + hash 链校验 |
| D-GATE-02 | Transaction Orchestrator 与 Saga/Workflow 边界 | EBCX-EV2-005/006 | ✅ Local ACID + Domain Event + Saga + Compensation + Evidence 五层组合 |
| D-GATE-03 | RPO ≤ 0 技术实现条件 | EBCX-DEPLOY-001 | ✅ T0~T3 分级表 + RPO≤0 仅适用 Core Transaction |
| D-GATE-04 | B1 Benchmark Profile 绑定 | EBCX-EV1-016/023、EBCX-EV2-016 | ✅ B1-EBCX-BASELINE Hardware+Workload 绑定 |
| D-GATE-05 | Graph Projection ≤3s 适用条件与指标 | EBCX-EV1-005/015 | ✅ Normal Mode P95≤3s + 模式分级 + 6 项指标 + DLQ |
| D-GATE-06 | 24 Entity + 8 Edge Canonical Graph Contract | EBCX-EV1-014 | ✅ Node/Edge Contract 锁死 + 边必由 Domain Event 驱动 |
| D-GATE-07 | Agent Execution Authorization 模型 | EBCX-EV2-013 | ✅ Agent→Reason→Policy→Approval→Authorize→Execute→Verify→Evidence + Independent Verifier |
| D-GATE-08 | 13/14/6 五层层级关系 | EBCX-EV1-001 | ✅ 五层层级体系 + 13→14 差异来源明确 |

**一致性校验结论**：spec.md §4.1~§5.13 全部 18 个章节 + design.md D01~D24 全部 24 项 + D-GATE-01~08 全部 8 项 Architecture Hardening 均被任务覆盖，无遗漏，无矛盾，无漂移。

---

# 八、10 条施工红线遵守情况

| 红线 | 内容 | 遵守任务 | 遵守情况 |
|---|---|---|---|
| TASK-R01 | 不得重新设计架构 | 全部任务（严格遵循 spec.md v1.1 + design.md v1.1 冻结基线） | ✅ Tasks 是 Design → Implementation Tasks 转化，未重新思考架构 |
| TASK-R02 | 不得把 14 Modules 拆成大量微服务 | EBCX-EV1-001（Modular Monolith + 14 模块 Go package 边界） | ✅ 第一阶段继续 Modular Monolith + Event-Native |
| TASK-R03 | Transaction Core 必须保持 Domain Boundary | EBCX-EV2-001~004（聚合根边界）+ EBCX-EV2-005（Orchestrator 编排） | ✅ 禁止 OrderService 直接修改 PaymentService 数据库，聚合根间通过 Domain Event + Orchestrator 编排 |
| TASK-R04 | Evidence Ledger 必须从第一批代码开始执行不可篡改模型 | EBCX-EV1-003（第一批基础设施任务） | ✅ 从第一个 Task 起，四层纵深防御 + Runtime Role 无 UPDATE/DELETE + append-only + hash 链 + correlation_id/causation_id 一并落地 |
| TASK-R05 | 所有重要 Mutation 必须进入 Evidence/Event/Outbox 治理链 | EBCX-EV1-004（Outbox）+ EBCX-EV1-020（CQRS 分离） | ✅ Mutation 走完整治理链；Read/Query 不强制走 Mutation Pipeline（CQRS 分离） |
| TASK-R06 | Neo4j 必须是 Projection | EBCX-EV1-005（Graph Projection）+ EBCX-EV1-014（边由 Domain Event 驱动） | ✅ 禁止业务代码直接把 Neo4j 当 Source of Truth，Neo4j 只通过 Outbox+EventBus 异步投影写入 |
| TASK-R07 | Agent Execution 必须经过 Policy/Authorization | EBCX-EV2-012/013（三重治理 + Authorization 模型） | ✅ Agent→Reason→Policy→Approval→Authorize→Execute→Verify→Evidence，不能自授执行权限，Independent Verifier 非自证 |
| TASK-R08 | B1 性能指标必须有 Benchmark Harness | EBCX-EV1-016（Benchmark Harness 框架）+ EBCX-EV1-023/016（B1 执行） | ✅ Load Generator + Dataset + Scenario + Measurement + Report + Evidence 全部落地 |
| TASK-R09 | 所有关键任务必须 Evidence First | 全部关键任务（含 Unit Test → Integration Test → Physical Verification → Evidence → Gate） | ✅ 每个关键任务含 Evidence 产出，Physical Verification 必须产生 Physical Evidence |
| TASK-R10 | 最终必须形成 Digital Engineering 闭环 | EBCX-REVIEW-003（变更确认） | ✅ Requirement → Design Decision → Task → Code → Test → Evidence → Verification → Closure 闭环验证 |

**10 条施工红线遵守结论**：TASK-R01~R10 全部 10 条施工红线均被任务严格遵守，无违反。

---

# 九、任务统计

## 9.1 任务总数

| 阶段 | 任务数 | 类型 |
|---|---|---|
| EV1 Enterprise Core | 24 | 细化（1~3 人日/任务） |
| EV2 Transaction Core | 17 | 细化（1~3 人日/任务） |
| EV3~EV12 | 10 | 里程碑级 |
| 集成测试 | 3 | 验证 |
| 部署配置 | 2 | 部署 |
| 评审验证 | 3 | 评审 |
| **合计** | **59** | — |

## 9.2 EV1 细化任务清单（24 个）

```text
EBCX-EV1-001 搭建 Modular Monolith 项目脚手架
EBCX-EV1-002 建立 PostgreSQL schema 基础
EBCX-EV1-003 实现 Evidence Ledger 基础设施（四层防御）🔴
EBCX-EV1-004 实现 Outbox + EventBus 基础设施
EBCX-EV1-005 实现 Neo4j Graph Projection 基础设施 🔴
EBCX-EV1-006 实现 Object Storage Artifact 基础设施
EBCX-EV1-007 实现 HTKIS-AF 安全基座
EBCX-EV1-008 实现多租户基础
EBCX-EV1-009 实现 Enterprise 聚合根
EBCX-EV1-010 实现 Organization 聚合根
EBCX-EV1-011 实现 Person 聚合根
EBCX-EV1-012 实现 MasterData 聚合根
EBCX-EV1-013 实现 Permission 聚合根
EBCX-EV1-014 实现 24 Entity Graph Schema + Node/Edge Contract 🔴
EBCX-EV1-015 实现 Observability 基础
EBCX-EV1-016 实现 Benchmark Harness 框架 🔴
EBCX-EV1-017 实现 REST API /api/v1/rel/* 第一入口
EBCX-EV1-018 实现 GraphQL 适配层
EBCX-EV1-019 实现 Event Contract + Schema Registry
EBCX-EV1-020 实现 CQRS Command/Query 分离基础
EBCX-EV1-021 实现 DevSecOps CI/CD 流水线
EBCX-EV1-022 实现 IaC 基础设施
EBCX-EV1-023 执行 B1 Benchmark 首次压测 + Measured Baseline 🔴
EBCX-EV1-024 EV1 Gate 评审准备
```

## 9.3 EV2 细化任务清单（17 个）

```text
EBCX-EV2-001 实现 Order 聚合根
EBCX-EV2-002 实现 Contract 聚合根
EBCX-EV2-003 实现 Invoice 聚合根
EBCX-EV2-004 实现 Payment 聚合根
EBCX-EVE-005 实现 Transaction Orchestrator 11 阶段编排 🔴
EBCX-EV2-006 实现 Saga/Workflow 跨服务编排 + Compensation
EBCX-EV2-007 实现 Domain Event 契约 + Schema Registry
EBCX-EV2-008 实现 CQRS Read Model 投影
EBCX-EV2-009 实现 Evidence Provenance & Data Lineage
EBCX-EV2-010 实现 Policy Engine 基础
EBCX-EV2-011 实现 Approval 聚合根（一等公民）
EBCX-EV2-012 实现 Agent Runtime 基础（三重治理）
EBCX-EV2-013 实现 Agent Execution Authorization 模型 🔴
EBCX-EV2-014 实现 REST API Transaction/Evidence/Policy/Agent/Graph
EBCX-EV2-015 实现 B2~B5 Profile 预留框架
EBCX-EV2-016 执行 B1 Benchmark + Measured Baseline 🔴
EBCX-EV2-017 EV2 Gate 评审准备
```

## 9.4 关键任务标记（🔴）

以下 8 个任务为关键任务，对应 10 条施工红线的核心落地：
- EBCX-EV1-003 Evidence Ledger 四层防御（TASK-R04）
- EBCX-EV1-005 Neo4j Graph Projection（TASK-R06）
- EBCX-EV1-014 24 Entity Graph Contract（TASK-R06）
- EBCX-EV1-016 Benchmark Harness（TASK-R08）
- EBCX-EV1-023 B1 Benchmark 首次执行（TASK-R08/R10）
- EBCX-EV2-005 Orchestrator 11 阶段编排（TASK-R03/R05）
- EBCX-EV2-013 Agent Authorization 模型（TASK-R07）
- EBCX-EV2-016 B1 Benchmark 含 Transaction Core（TASK-R08/R10）

---

# 十、Digital Engineering 闭环验证

```text
Requirement（spec.md v1.1）
    ↓
Design Decision（design.md v1.1，D01~D24 + D-GATE-01~08）
    ↓
Task（tasks.md v1.0，59 个任务）
    ↓
Code（EV1 + EV2 实现）
    ↓
Test（Unit Test + Integration Test + E2E Test）
    ↓
Evidence（Physical Evidence，每个任务产出）
    ↓
Verification（Physical Verification + B1 Benchmark + 一致性校验）
    ↓
Closure（EV1 Gate + EV2 Gate + 评审验证）
```

**闭环验证结论**：Digital Engineering 闭环完整，每个环节均有 Physical Evidence 产出，可追溯、可审计、可验证。

---

## 文档结束

> 本 tasks.md v1.0 基于 spec.md v1.1（EV0-SPEC PASS / CLOSED / 🔒 FROZEN）+ design.md v1.1（EV0-DESIGN PASS / CLOSED / 🔒 FROZEN）生成，覆盖 EV1~EV12 全部阶段任务，重点细化 EV1 Enterprise Core（24 个任务）与 EV2 Transaction Core（17 个任务），EV3~EV12 为里程碑级任务（10 个）。
>
> **任务统计**：59 个任务（EV1: 24 + EV2: 17 + EV3~EV12: 10 + 集成测试: 3 + 部署配置: 2 + 评审验证: 3）。
>
> **一致性校验**：spec.md §4.1~§5.13 全部 18 个章节 + design.md D01~D24 全部 24 项 + D-GATE-01~08 全部 8 项 Architecture Hardening 均被任务覆盖，无遗漏，无矛盾，无漂移。
>
> **10 条施工红线**：TASK-R01~R10 全部 10 条施工红线均被任务严格遵守，无违反。
>
> **8 条架构红线 + 7 条母架构约束**：全部对齐，无漂移。
>
> **Digital Engineering 闭环**：Requirement → Design Decision → Task → Code → Test → Evidence → Verification → Closure 闭环完整。
>
> **后续阶段**：EV1 编码须在 EV0-G0 Gate 通过后方可启动。本 tasks.md v1.0 待大G项目经理体系 EV0-TASKS Gate 复审。