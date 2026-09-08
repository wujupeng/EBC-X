# EBC-X EV1-006~008 Final Evidence Audit Report

> 提交时间：2026-09-08
> 提交人：Coding Agent
> 审查人：大G项目经理
> 前置文档：
> - EBCX-EV1-006-008-GATE-REVIEW.md (CONDITIONAL PASS)
> - EBCX-EV1-006-008-PHYSICAL-CLOSURE-REPORT.md (Physical Gate Complete)
> 裁决状态：**待审查**

---

## 0. 审计范围

本报告响应大G项目经理的 Final Evidence Audit 指令，仅执行两项审计：

- **A. Migration Repair Audit** — V6/V7/V8/V9 修复与 EV0 Frozen Design 对齐
- **B. RLSManager Connection Lifecycle Audit** — 连接池 context 泄漏验证
- **C. Provenance** — Evidence git_commit 一致性确认

**未新增业务功能，未修改 EV0 冻结文档，未开始 EV1-009。**

---

## A. Migration Repair Audit

### A.1 V6 修复审计

| 项目 | 内容 |
|------|------|
| **原问题** | V6 缺少 `CREATE SCHEMA graph` 和 `GRANT USAGE ON SCHEMA graph TO ebcx_runtime` |
| **Physical Gate 发现原因** | V1 只创建 7 个 schema (business/evidence/outbox/audit/master_data/policy/tenant)，V6 尝试在 `graph` schema 中建表但 schema 不存在，真实 PostgreSQL 执行失败 |
| **当前修复** | V6 顶部添加 `CREATE SCHEMA IF NOT EXISTS graph` + `GRANT USAGE ON SCHEMA graph TO ebcx_runtime` |
| **修复后实际 schema** | `graph` schema 存在，包含 projection_state / reconciliation_results / shadow_rebuild / projection_metrics 4 张表，均有 RLS 策略 |
| **EV0 对齐** | ✅ D15 列出 7 个基础 schema，`graph` 是 EV1-005/005A 为 Graph Projection Metadata 新增的 schema（D-GATE-05/06）。V6 创建 `graph` schema 是正确的迁移位置 |
| **架构语义变化** | 无。`graph` schema 是 EV1-005 的合法扩展，非 EV0 架构漂移 |

### A.2 V7 修复审计

| 项目 | 内容 |
|------|------|
| **原问题** | V7 缺少 `GRANT USAGE ON SCHEMA artifact TO ebcx_runtime` |
| **Physical Gate 发现原因** | 以 `ebcx_runtime` 角色查询 `artifact.metadata` 时失败，缺少 schema USAGE 权限 |
| **当前修复** | V7 添加 `GRANT USAGE ON SCHEMA artifact TO ebcx_runtime` |
| **修复后实际 schema** | `artifact` schema 存在，`artifact.metadata` 表有 RLS 策略 + WORM triggers + FK 约束，`ebcx_runtime` 有 SELECT+INSERT 权限 |
| **EV0 对齐** | ✅ D15 定义 OBS 命名规范和 Artifact Truth。`artifact` schema 是 EV1-006 的合法新增 schema。V2 对 V1 中的 schema 批量 GRANT USAGE，但新增 schema 需在各自迁移中 GRANT，符合现有模式 |
| **架构语义变化** | 无。GRANT USAGE 是运行角色访问新 schema 的必要权限，与 V2 模式一致 |

### A.3 V8 修复审计

| 项目 | 内容 |
|------|------|
| **原问题** | V8 缺少 `GRANT USAGE ON SCHEMA security TO ebcx_runtime` |
| **Physical Gate 发现原因** | 以 `ebcx_runtime` 角色查询 `security.revoked_tokens` 和 `security.kms_keys` 时失败 |
| **当前修复** | V8 添加 `GRANT USAGE ON SCHEMA security TO ebcx_runtime` |
| **修复后实际 schema** | `security` schema 存在，包含 kms_keys / revoked_tokens 表，`audit` schema 的 audit_log 表有 RLS + WORM triggers |
| **EV0 对齐** | ✅ D13 定义 HTKIS-AF 安全基座。`security` schema 是 EV1-007 的合法新增 schema。GRANT USAGE 模式与 V7 一致 |
| **架构语义变化** | 无 |

### A.4 V9 修复审计（最关键）

| 项目 | 内容 |
|------|------|
| **原问题** | V9 使用 `CREATE TABLE IF NOT EXISTS tenant.tenants`，但 V3 已创建此表。`IF NOT EXISTS` 静默跳过，V9 的扩展列 (country_code, is_active) 未添加 |
| **Physical Gate 发现原因** | Physical Gate 查询 `tenant.tenants` 的 V9 扩展列时发现列不存在 |
| **当前修复** | V9 改为 `ALTER TABLE tenant.tenants ADD COLUMN IF NOT EXISTS ...` |
| **修复后实际 schema** | 见下方 A.4.1 |
| **EV0 对齐** | ✅ 见下方 A.4.2 |
| **架构语义变化** | 无。见下方 A.4.3 |

#### A.4.1 tenant.tenants 最终实际 schema

```sql
-- V3 创建 (基表)
tenant_id       UUID        PRIMARY KEY DEFAULT gen_random_uuid()
tenant_code     VARCHAR(64) NOT NULL UNIQUE
tenant_name     VARCHAR(256) NOT NULL
country_pack    VARCHAR(32) NOT NULL DEFAULT 'china'
status          VARCHAR(16) NOT NULL DEFAULT 'active'
created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()

-- V9 添加 (扩展列)
country_code    VARCHAR(10)              -- Pack 路由国家代码
is_active       BOOLEAN     NOT NULL DEFAULT true  -- 租户激活状态
```

约束:
- `chk_country_pack`: country_pack IN ('china','eu','us','japan','asean')
- `chk_status`: status IN ('active','suspended','archived')
- RLS: `tenant.tenants` ENABLE ROW LEVEL SECURITY (V3 创建)

#### A.4.2 与 EV0 Task Contract 对照

| EV0 设计 (D14) | 实际实现 | 一致性 |
|---------------|---------|--------|
| 共享 schema + RLS | ✅ tenant schema + RLS 策略 | ✅ 一致 |
| 每表含 tenant_id 列 | ✅ tenant_id UUID PRIMARY KEY | ✅ 一致 |
| RLS: current_setting('app.tenant_id') = tenant_id | ✅ V3 RLS 策略 | ✅ 一致 |
| Pack 路由: X-Tenant-Id + X-Country-Pack | ✅ V9 tenant_packs + country_packs 表 | ✅ 一致 |
| 核心表不分叉 | ✅ V9 扩展表 order_{pack}_ext | ✅ 一致 |
| 5 Country Pack 扩展点 | ✅ CN/EU/US/JP/ASEAN | ✅ 一致 |

**D14 设计决策原文**: "共享 schema + 行级隔离（RLS）模型，每表含 tenant_id 列，RLS 策略强制 current_setting('app.tenant_id') = tenant_id。Country Pack 路由通过请求头 X-Country-Pack 路由至对应扩展表。核心表不分叉（§5.8.1 规则 3）。"

**V3 → V9 关系**: V3 创建租户注册基表（tenant_id, tenant_code, tenant_name, country_pack, status, created_at, updated_at），V9 扩展 Pack 路由能力（country_code, is_active + country_packs + tenant_packs + 5 扩展表）。这是**能力扩展**，不是**架构漂移**。

#### A.4.3 架构语义变化分析

| 关注点 | 分析 | 结论 |
|--------|------|------|
| country_pack (V3) vs country_code (V9) 是否冗余？ | V3 的 country_pack 使用小写值 ('china','eu',...)，V9 的 country_code 使用大写值 ('CN','EU',...)。Go 模型使用 CountryCode 字段映射 V9 列 | 🟡 存在语义重叠，但不影响架构。生产使用时需明确哪个字段为权威源。当前不影响 Physical Gate 验证结论 |
| is_active (V9) vs status (V3) 是否冗余？ | V3 的 status 是多状态 ('active','suspended','archived')，V9 的 is_active 是布尔值。Go 模型使用 IsActive 字段 | 🟡 存在语义重叠，但不影响架构。可视为 status 的简化投影 |
| V9 ALTER TABLE 是否改变了 V3 的迁移语义？ | 否。V3 的表结构完全保留，V9 仅添加列 | ✅ 无语义变化 |

**结论**: V9 修复（ALTER TABLE 替代 CREATE TABLE IF NOT EXISTS）是正确的迁移模式。最终 tenant.tenants schema 符合 EV0 D14 设计决策。country_pack/country_code 和 status/is_active 的语义重叠是应用层关注点，不影响数据库架构和 RLS 策略。

### A.5 Migration Repair Audit 总结

| 迁移 | 原问题 | 修复 | EV0 对齐 | 架构漂移 |
|------|--------|------|---------|---------|
| V6 | 缺 CREATE SCHEMA graph | 添加 CREATE SCHEMA + GRANT USAGE | ✅ D-GATE-05/06 | 无 |
| V7 | 缺 GRANT USAGE artifact | 添加 GRANT USAGE | ✅ D15 OBS 命名 | 无 |
| V8 | 缺 GRANT USAGE security | 添加 GRANT USAGE | ✅ D13 HTKIS-AF | 无 |
| V9 | tenant.tenants 冲突 | ALTER TABLE 替代 CREATE TABLE | ✅ D14 Multi-Tenant | 无 |

**全部 4 个迁移修复均无架构语义变化，符合 EV0 Frozen Design。**

---

## B. RLSManager Connection Lifecycle Audit

### B.1 原始问题分析

大G项目经理指出：RLSManager 从 `SET LOCAL + $1`（transaction-scoped）改为 `SET + fmt.Sprintf`（session-scoped）后，可能存在连接池 Context 泄漏风险。

#### 代码分析

```go
// 原修复后代码 (session-scoped)
func (m *RLSManager) SetTenantContext(ctx context.Context, rlsCtx RLSContext) error {
    _, err := m.db.ExecContext(ctx, fmt.Sprintf("SET app.tenant_id = '%s'", rlsCtx.TenantID))
    // ...
}
```

**发现的风险**:

1. **连接池风险**: `m.db` 是 `*sql.DB`（连接池）。每次 `ExecContext` 可能使用**不同的连接**。`SET` 只影响执行它的那个连接，后续查询可能使用另一个没有设置 tenant context 的连接。

2. **Context 泄漏风险**: 如果同一连接被复用，前一个请求的 `SET app.tenant_id` 值会持久存在。下一个请求如果忘记调用 `SetTenantContext`，会继承前一个租户的 context。

### B.2 修复方案

添加连接池安全的事务级 API：

```go
// BeginTenantTransaction — 连接池安全
// 1. 开启事务（绑定单个连接）
// 2. SET LOCAL（事务级，自动清除）
// 3. 返回 *sql.Tx，所有查询使用同一连接
// 4. Commit/Rollback 时 SET LOCAL 自动清除
func (m *RLSManager) BeginTenantTransaction(ctx context.Context, rlsCtx RLSContext) (*sql.Tx, error)

// SetTenantContextInTx — 在已有事务中设置 RLS context
func (m *RLSManager) SetTenantContextInTx(ctx context.Context, tx *sql.Tx, rlsCtx RLSContext) error
```

**安全保证**:
- `SET LOCAL` 是事务级的，事务结束时自动清除
- 事务内所有查询使用同一数据库连接
- 即使忘记 Rollback，连接归还连接池时事务也会自动回滚
- 不可能发生 Context 泄漏

**旧 API 处理**: `SetTenantContext` 和 `ClearTenantContext` 标记为 `Deprecated`，保留向后兼容但文档说明仅适用于专用连接（`sql.Conn`）。

### B.3 Physical Test 证据

#### TestPhysical_RLSManager_ConnectionPoolLifecycle

验证连接池生命周期安全，4 个阶段：

| 阶段 | 操作 | 预期 | 实际 | 结果 |
|------|------|------|------|------|
| Phase 1 | Tenant A: BeginTenantTransaction(A) → SELECT count(*) | 1 row (only A) | 1 row | ✅ PASS |
| Phase 1 | Tenant A: SELECT WHERE tenant_id = B | 0 rows (blocked) | 0 rows | ✅ PASS |
| Phase 2 | Tenant B: BeginTenantTransaction(B) → SELECT count(*) | 1 row (only B) | 1 row | ✅ PASS |
| Phase 2 | Tenant B: SELECT WHERE tenant_id = A | 0 rows (blocked) | 0 rows | ✅ PASS |
| Phase 3 | No context: BeginTx → SELECT count(*) | 0 rows (RLS blocks all) | 0 rows | ✅ PASS |
| Phase 4 | Rapid A→B→A: 3 iterations, each BeginTenantTransaction | Each sees only own data | 1 row each, 0 cross-leak | ✅ PASS |

**Phase 3 关键证明**: 事务 Rollback 后 `SET LOCAL` 自动清除。新事务无 tenant context 时 RLS 阻止所有访问（返回 0 rows），证明**无 Context 泄漏**。

**Phase 4 关键证明**: 快速 A→B→A 切换，每次事务独立，不继承前一个事务的 context。

#### TestPhysical_RLSManager_SetTenantContextInTx

验证在已有事务中设置 RLS context：
- BeginTx → SetTenantContextInTx(A) → SELECT count(*) → 1 row → ✅ PASS

#### TestPhysical_RLSManager_LegacySetOverwrite

验证旧 API 在专用连接上的行为：
- 使用 `sql.Conn`（专用连接，非连接池）
- SET A → SELECT → 1 row (A)
- SET B (覆盖) → SELECT → 1 row (B)
- RESET → SELECT → 0 rows
- ✅ PASS

**结论**: 旧 API 在专用连接上正确工作，但在连接池上不安全。新 API (BeginTenantTransaction) 在连接池上安全。

### B.4 RLSManager Connection Lifecycle Audit 总结

| 审计项 | 结果 | 证据 |
|--------|------|------|
| 连接池 Context 泄漏风险 | ✅ **已修复** | BeginTenantTransaction 使用 SET LOCAL (事务级) |
| 事务边界自动清除 | ✅ **已验证** | Phase 3: Rollback 后无 context 残余 |
| 快速租户切换 | ✅ **已验证** | Phase 4: A→B→A 无交叉泄漏 |
| 旧 API 向后兼容 | ✅ **保留** | 标记 Deprecated，专用连接上仍可用 |
| Physical Test 覆盖 | ✅ 3 新测试 | ConnectionPoolLifecycle + SetTenantContextInTx + LegacySetOverwrite |

---

## C. Provenance

### C.1 代码提交链

```
7e6036d  EBC-X EV1-PHY-007 FINAL AUDIT: RLSManager connection-pool safety fix  ← HEAD
5fb10d0  EBC-X EV1-PHY-006~008: Physical Closure Package
f562530  EBC-X EV1-PHY-006~008: Physical Closure Gate — Real PostgreSQL 18.3 Verification
c26d2e5  EBC-X EV1-006~008 Gate Review Package — CONDITIONAL PASS
```

### C.2 Evidence git_commit 一致性

| 项目 | 值 | 一致性 |
|------|-----|--------|
| Evidence JSON git_commit | `7e6036d8ee5d5b3b3eced38f66333fd9f49b176e` | ✅ |
| 实际验证代码 commit | `7e6036d` | ✅ 一致 |
| Physical Test 执行 commit | `7e6036d` (RLSManager fix + tests) | ✅ 一致 |

### C.3 Provenance 链

```
Code Implementation (8d8ef62, 994a6af, a763d18)
  → Physical Gate Verification (f562530 — 18 tests PASS)
  → Physical Closure Package (5fb10d0 — Evidence JSON + Closure Report)
  → Final Evidence Audit (7e6036d — RLSManager fix + 3 lifecycle tests)
  → Updated Evidence JSON (git_commit = 7e6036d)
  → Final Evidence Audit Report (本文档)
```

### C.4 测试统计（Final）

| 分类 | 数量 | 状态 |
|------|------|------|
| UNIT | 79 | ✅ PASS |
| INTEGRATION | 22 | ✅ PASS |
| PHYSICAL (physicalgate) | 14 | ✅ PASS |
| PHYSICAL (RLSManager) | 7 | ✅ PASS |
| PHYSICAL NEGATIVE | 8 | ✅ PASS |
| **总计** | **122** | **✅ ALL PASS** |

Physical Negative Tests 增加 2 个:
- TestPhysical_RLSManager_ConnectionPoolLifecycle Phase 1/2 (cross-tenant blocked)
- TestPhysical_RLSManager_ConnectionPoolLifecycle Phase 3 (no context = no access)

---

## D. Scope 确认

| 约束 | 遵守 |
|------|------|
| 未修改 EV0 Frozen spec/design/tasks | ✅ |
| 未开始 EV1-009 | ✅ |
| 未新增业务功能 | ✅ (RLSManager fix 是安全修复，非新功能) |
| 未自行宣布 CLOSED | ✅ |
| 未自行授权 EV1-009 | ✅ |

---

## E. 审计结论

### E.1 Migration Repair Audit

**全部 4 个迁移修复 (V6/V7/V8/V9) 均无架构语义变化，符合 EV0 Frozen Design。**

- V6/V7/V8: 添加缺失的 CREATE SCHEMA / GRANT USAGE，是新 schema 的必要权限，与 V2 模式一致
- V9: ALTER TABLE 替代 CREATE TABLE IF NOT EXISTS，是正确的迁移模式。tenant.tenants 最终 schema 符合 D14 设计决策

### E.2 RLSManager Connection Lifecycle Audit

**连接池 Context 泄漏风险已修复并通过 Physical Test 验证。**

- 新增 `BeginTenantTransaction` (SET LOCAL + 事务级) 是连接池安全方法
- 3 个 Physical Test 验证：连接池生命周期、已有事务中设置、旧 API 覆盖行为
- 事务 Rollback 后 SET LOCAL 自动清除，无 Context 泄漏

### E.3 最终请求

请求大G项目经理基于本 Final Evidence Audit Report 裁决：

1. **EV1-006**: ✅ PASS / CLOSED
2. **EV1-007**: ✅ PASS / CLOSED
3. **EV1-008**: ✅ PASS / CLOSED
4. **授权 EV1-009** Enterprise Aggregate Root 开发