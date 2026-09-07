# EBC-X

> **Enterprise Business & Industrial Operating System**
> **企业与工业智能运营操作系统**

[![Status](https://img.shields.io/badge/EV0-SPEC-PASS%20%2F%20CLOSED-brightgreen)]()
[![Design](https://img.shields.io/badge/EV0-DESIGN-v1.1%20PASS%20%2F%20CLOSED-brightgreen)]()
[![Tasks](https://img.shields.io/badge/EV0-TASKS-v1.1%20HARDENED-yellow)]()

---

## 一句话定位

**EBC-X 不是下一代 ERP。**

EBC-X 是面向制造业和中大型企业的 **Enterprise Business & Industrial Operating System** —— 以业务交易为核心、以工业数据为基础、以 Evidence 为信任底座、以 AI Agent 为执行层、以 Digital Twin 为预测与验证层的企业与工业智能运营操作系统。

```
EBC-X = Business Operating System
      + Industrial Data Platform
      + Evidence Graph
      + Governed Agent
      + Digital Twin
```

---

## 战略定位

```
传统 ERP:  Data → Workflow → Report

EBC-X:     Business State
         + Event
         + Evidence
         + Graph
         + Policy
         + Agent
         + Simulation
```

> **EBC-X ≠ SAP / Oracle 国产复刻**（永久架构原则）

借鉴成熟 ERP 最佳实践，但核心创新在于 Evidence-First + Governed Agentic ERP。

---

## 治理模型

| 角色 | 核心责任 |
|------|---------|
| **大G项目经理体系** | 总体战略、产品定义、架构裁决、阶段 Gate |
| **HTKIS** | 产品主权、知识产权、品牌、全球交付、实施、客户成功 |
| **华为云团队** | 核心软件研发、云原生工程、基础设施适配、DevSecOps |

> 华为云负责"做出来"。HTKIS 负责"交付出去"。大G 负责"决定做什么、做到什么标准、什么时候算完成"。

---

## 母架构

```
                         ┌──────────────────────────┐
                         │        EBC-X              │
                         │ Enterprise Business OS    │
                         └────────────┬─────────────┘
                                      │
              ┌───────────────────────┼───────────────────────┐
              ↓                       ↓                       ↓
       Enterprise Core        Transaction Core        Industrial Core
              │                       │                       │
              └───────────────────────┼───────────────────────┘
                                      ↓
                           Enterprise Event Backbone
                                      │
                    ┌─────────────────┼─────────────────┐
                    ↓                 ↓                 ↓
             Evidence Ledger    Graph Projection    Artifact Store
              PostgreSQL           Neo4j              Object Storage
                    │                 │                 │
                    └─────────────────┼─────────────────┘
                                      ↓
                            Data / Lineage / Search
                                      │
                     ┌────────────────┼────────────────┐
                     ↓                ↓                ↓
                Policy Engine    Agent Runtime    Digital Twin
                     │                │                │
                     └────────────────┼────────────────┘
                                      ↓
                              Verification
                                      ↓
                                  Evidence
                                      ↓
                                  Closure
```

### 母架构约束（7 条，永久锁定）

1. **PostgreSQL Evidence/Transaction Truth 是根**
2. **Neo4j 是 Graph Query Projection**（可挂、可慢、可重建、可清空后重新 Projection）
3. **Object Storage 是 Artifact Truth**
4. **EventBus 是传播机制，不是真实性来源**
5. **Agent 不能越过 Policy**
6. **Policy 不能越过 Authorization**
7. **Execution 必须产生 Evidence**

---

## 三层真相模型

| 层 | 技术 | 职责 | 语义 |
|----|------|------|------|
| **Evidence Truth** | PostgreSQL | 原始 Evidence、交易事实、审计事实、版本、租户隔离 | System of Record（append-only） |
| **Graph Query Truth** | Neo4j | 图关系、路径、子图、关系推理、Policy 查询、Agent 上下文 | Graph Projection（异步，可重建） |
| **Artifact Truth** | Object Storage | PDF、图片、CAD、质检报告、发票原件、合同附件 | Immutable（WORM） |

```
Transaction Core → PostgreSQL (Evidence Ledger)
                        │
                     Outbox
                        ↓
                    EventBus
                        ↓
              Neo4j (Graph Projection)  ← 异步，最终一致
```

> Graph Projection 可以失败、延迟、重建，但 Transaction Core 不得因此失败。

---

## 第一性原理链路

```
Business
   ↓
Event
   ↓
Transaction
   ↓
Data
   ↓
Evidence
   ↓
Policy
   ↓
Decision
   ↓
Agent
   ↓
Execution
   ↓
Verification
   ↓
Closure
```

**适用范围**：仅"业务状态变更、受治理决策及自动化执行行为"（Mutation Path）走完整 Governed Execution Chain。Read / Query 路径分流，不强制走完整 11 阶段链路。

---

## 三大核心与既有项目归并

| 核心 | 归并自 | 职责 |
|------|--------|------|
| **Business Core** | EITP | Transaction Core |
| **Industrial Core** | AirPLM / MES + SeaFusion-X | Manufacturing + Digital Twin Validation |
| **Trust Core** | AeroForge-X + HTKIS-AF + HTKIZ/HyperDisk | Evidence Governance + Security + Storage |

---

## Evidence Graph（24 类 Canonical Entity）

```
Enterprise    Organization  Person       Product       Material
Supplier      Customer      Order        Contract      Invoice
Payment       Production    Quality      Asset         Project
Patent        R&D           Data         Evidence      Event
Policy        Decision      Agent        Approval
```

- **Node Contract**：NodeType / EntityId / TenantId / Version / Status / Source / EvidenceRefs ...
- **Edge Contract**：EdgeType / FromEntity / ToEntity / Validity / SourceEvent / EvidenceRef ...
- **边必由 Domain Event 驱动创建，禁止 Graph AI / heuristic 推断关系**

---

## Governed Agent（受治理智能体）

```
Agent → Reason → Policy → Approval → Authorize → Execute → Verify → Evidence
```

- **Agent 不能自授执行权限**
- **Independent Verifier 非自证**（Agent 执行后由独立验证器校验）
- **Execution Token + Idempotency Key**
- **任何影响决策的关键事实必须可追溯 Evidence**

> **Governed Agentic ERP** — AI 可以建议、申请、执行，但企业制度决定它能不能执行，Evidence 决定它做了什么。

---

## 技术栈

| 层 | 选型 |
|----|------|
| **后端** | Go + Rust 双语言架构 |
| **数据库** | PostgreSQL（System of Record，RLS / append-only）+ Neo4j（Graph Projection） |
| **对象存储** | 华为云 OBS（Evidence Artifact Store，WORM） |
| **消息** | 华为云 DMS（Kafka 兼容）+ Outbox + 至少一次 + 幂等 |
| **API** | REST `/api/v1/rel/*`（第一入口）+ GraphQL（适配层） |
| **架构风格** | DDD（聚合根 + Orchestrator 编排）+ CQRS + Event-Native + Modular Monolith |
| **前端** | React + Antd + Vite |
| **部署** | 华为云 CCE（容器）+ DevSecOps + IaC + 多 AZ |

> **第一阶段：Modular Monolith + Event-Native，而不是 Microservice Showcase。**

---

## 五层层级体系

```
Layer 1 — Core Capabilities:     13 capabilities（业务能力）
Layer 2 — Domain Modules:        14 implementation modules（含 1 Platform/Governance）
Layer 3 — Bounded Contexts:      6 contexts（限界上下文）
Layer 4 — Canonical Entities:    24 entities（Evidence Graph 节点）
Layer 5 — Cross-cutting Platform: Event / Evidence / Policy / Identity / Observability
```

---

## 路线图

| 阶段 | 名称 | 核心成果 | 状态 |
|------|------|---------|------|
| **EV0** | Architecture & Policy | 总体架构 | 🟢 PASS / CLOSED |
| **EV1** | Enterprise Core | 企业/组织/主数据/权限 | 🟡 TASKS READY |
| **EV2** | Transaction Core | EITP → EBC-X | ⚪ PLANNED |
| **EV3** | Evidence Core | Evidence Graph | ⚪ PLANNED |
| **EV4** | Finance | 财务核心 | ⚪ PLANNED |
| **EV5** | SCM | 供应链 | ⚪ PLANNED |
| **EV6** | Manufacturing | MES/生产 | ⚪ PLANNED |
| **EV7** | PLM/Quality/EAM | 工业核心 | ⚪ PLANNED |
| **EV8** | Data Trust | 工业数据基础设施 | ⚪ PLANNED |
| **EV9** | Agent Runtime | 企业智能体 | ⚪ PLANNED |
| **EV10** | Digital Twin | 企业数字孪生 | ⚪ PLANNED |
| **EV11** | Industry Network | 产业链数据协同 | ⚪ PLANNED |
| **EV12** | Global Platform | 全球化商业化 | ⚪ PLANNED |

---

## 全球化架构

```
EBC-X Core
   │
   ├── China Pack   (RMB / 中国会计准则 / VAT / 发票 / 国产化)
   ├── EU Pack      (EUR / IFRS / GDPR / VAT)
   ├── US Pack      (USD / US GAAP / Tax)
   ├── Japan Pack   (JPY / J-GAAP)
   └── ASEAN Pack   (Singapore / Vietnam / Thailand)
```

> Global Core + Country Pack，第一期不实现所有国家。

---

## 工信部 1+4+N 产品能力映射

| 工信部方向 | EBC-X 对应能力 |
|-----------|---------------|
| **1：可信互联平台** | Enterprise Data Trust Platform |
| **数据资源库** | Enterprise Data Resource Lake |
| **数据技术库** | Data Technology Hub |
| **工业数据标准库** | Enterprise Data Standard Graph |
| **高质量数据集库** | Evidence Dataset Platform |
| **N：工业智能体** | Agent Runtime |

> **政策是产品设计的战略输入，不是产品需求规格本身。** EBC-X 能力架构与工信部《工业数据筑基行动》"1+4+N"体系进行产品能力映射。

---

## 架构红线（8 条）

1. ❌ 禁止第一期直接开发 1000+ 页面 ERP
2. ❌ 禁止先做"财务、采购、销售、库存、HR、CRM、OA"大杂烩
3. ❌ 禁止一开始拆成数百个微服务
4. ❌ 禁止为了 AI 而 AI
5. ❌ 禁止 Agent 绕过权限、政策和审批直接修改核心账务
6. ❌ 禁止形成厂商锁定架构
7. ❌ 禁止绕过 EV0 直接进入大规模业务编码
8. ❌ 禁止开发团队决定产品架构

---

## 项目文档

| 文档 | 路径 | 状态 | 说明 |
|------|------|------|------|
| **需求规格** | [`.codeartsdoer/specs/ebcx_ev0_arch/spec.md`](.codeartsdoer/specs/ebcx_ev0_arch/spec.md) | 🟢 v1.1 FROZEN | EARS 格式，1094 行，13 个能力模块 |
| **技术设计** | [`.codeartsdoer/specs/ebcx_ev0_arch/design.md`](.codeartsdoer/specs/ebcx_ev0_arch/design.md) | 🟢 v1.1 FROZEN | D01~D24 + D-GATE-01~08，约 2196 行 |
| **任务规划** | [`.codeartsdoer/specs/ebcx_ev0_arch/tasks.md`](.codeartsdoer/specs/ebcx_ev0_arch/tasks.md) | 🟡 v1.1 HARDENED | 60 个任务，EV1(25) + EV2(17) + EV3~EV12(10) + 集成/部署/评审(8)，TASK-H01~H10 加固 |

---

## Digital Engineering 闭环

```
Requirement → Design Decision → Task → Code → Test → Evidence → Verification → Closure
```

每个关键任务遵循：

```
Task → Implementation → Unit Test → Integration Test → Physical Verification → Evidence → Gate
```

---

## Benchmark Profile

```
B1-EBCX-BASELINE
  Hardware:  CPU / RAM / PostgreSQL / Kafka(DMS) / Redis / OpenSearch / Neo4j
  Workload:  50 tenants / 10000 users / 2000 concurrency
  Target:    TPS ≥ 2000 / P95 ≤ 500ms / Error ≤ 0.1%
```

B2~B5（Standard / Large / High-Concurrency / Extreme）预留框架。

---

## RPO 分级

| Tier | 数据 | RPO | 技术实现 |
|------|------|-----|---------|
| T0 | Core Transaction（Evidence Ledger） | ≤ 0 | 同步复制 + quorum commit + 多 AZ |
| T1 | Business Critical | ≤ 30s | 异步复制 + 同城灾备 |
| T2 | General Business | ≤ 5min | 异地灾备 |
| T3 | Analytics / Projection | profile-defined | Outbox + EventBus 最终一致，可重建 |

---

## License

Proprietary — HTKIS retains product sovereignty and intellectual property.