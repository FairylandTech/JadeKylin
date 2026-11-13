# Global Git Commit Instructions / 全局 Git 提交信息规范

概述 / Overview

- 目的：为所有由 AI 或团队成员提交到本组织/仓库的代码提供统一的提交信息规范，以便自动化（changelog、release、CI 检查）和审阅可读性。
- 适用范围：适用于本仓库及子项目，项目可在 repo 级创建覆盖文件（见“覆盖与例外”）。

语言选择 / Language choice

- 推荐原则：
    - 若仓库为公开或国际化项目，优先使用英文（English）以便全球协作者阅读和自动化工具兼容。
    - 若仓库为公司/团队内部且主要成员以中文为母语，使用中文完全可接受（但需在 README/CONTRIBUTING 里声明）。
    - 最佳折中：提交标题使用英文（简短、标准化），正文/详细说明可用中文补充本地化信息。关键是“保持一致性”。
- 建议：在仓库 README 或 CONTRIBUTING 中明确声明所选语言（例如 "Commit messages MUST be written in English" 或 "commit messages MAY be written in Chinese"）。

在标题最开头添加 Emoji（新）

- 规范要求：提交信息标题（单行 summary）必须以单个 emoji 开头，后接空格，再按格式书写 type (scope): short imperative summary。
- 目的：通过可视化符号快速在日志/PR 列表中识别变更类型，提升可读性与快速筛查效率。
- 要求：
    - emoji 放在标题开头的第一个字符位置，后跟一个空格，例如：✨ feat (auth): add JWT refresh token endpoint
    - emoji 应与 type 语义一致（见下表建议映射），但仓库可在项目级覆盖文件中替换为其他 emoji。
    - 标题长度与其他规则不变（建议 ≤ 50 字符摘要长度，不以句号结尾）。

提交信息格式 / Commit message format (推荐使用 Conventional Commits + emoji)
格式（单行标题 + 可选正文 + 可选 footer）：
<emoji> type (scope): short imperative summary (max 50 chars)

[optional body — describe motivation and why, wrap at 72 chars]

[optional footer — issue references, BREAKING CHANGE: description]

Emoji 与 type 建议映射（可调整）

- ✨ feat: 新功能
- 🐛 fix: 修复 bug
- 📝 docs: 文档变更
- 🎨 style: 不影响代码含义的样式（空格、格式化等）
- ♻️ refactor: 重构（既不是 fix 也不是 feat）
- ⚡ perf: 性能优化
- ✅ test: 新增或修改测试
- 🛠️ build: 构建系统或外部依赖的改动（例如 CI、Docker）
- ⚙️ ci: CI 配置相关改动
- 🔧 chore: 其他杂项（例如依赖更新，不修改 src）
- ⏪ revert: 回滚之前的提交
- 🚧 wip: 工作进行中（仅用于临时分支，不应合并到主分支）
- 💥 BREAKING CHANGE（ footer 中标记或在 PR 描述中强调）

示例 / Examples

- English (public project)
    - ✨ feat (auth): add JWT refresh token endpoint
    - 🐛 fix (api): validate pagination params to prevent crash
    - 📝 docs: update contributing guide with commit rules
- 双语示例（标题英文，正文中文）
    - ✨ feat (ui): add bulk-select for items
    -
    - 实现了批量选择功能，包含全选/反选；提高了列表操作效率。
    - Closes #321
- 中文示例（内部仓库）
    - 🐛 fix (storage): 修复对象存储上传失败的 race condition
    -
    - 修复了并发上传时出现的索引覆盖问题，增加了单元测试覆盖。
    - BREAKING CHANGE: 存储模块配置项 key_name 改为 storage_key

规则细节

- type：使用上表中的之一（小写）。
- scope：可选，表示受影响的模块或包（例如 auth, api, ui）。
- summary（标题）：使用祈使句（imperative），不要以句号结尾，长度建议 ≤ 50 字符（或根据仓库约定）。
- body：解释“为什么要改动”而非“做了什么”的每行约 72 字符换行；包含迁移、回滚或已知限制等。
- footer：用于关闭 issue（e.g. "Closes #123"）或记录 BREAKING CHANGE（推荐同时在 footer 使用文本说明并在 PR 标题/描述中显著说明）。

关于 AI 生成的提交

- AI 在生成提交信息时应：
    - 在标题最前面添加语义一致的 emoji。
    - 在 commit body 中明确说明改动范围与动机（至少在 commit body 中）。
    - 为重大或有风险的变更标注 BREAKING CHANGE 或风险说明。
    - 若生成代码包含外部依赖或凭证风险，commit message 必须在 body 中说明已执行的安全检查。
- 若 AI 不确定改动意图，应提出澄清性问题而不是编写模糊的提交信息。

自动化与检查 / Automation & enforcement

- 推荐工具（可选）：
    - commitlint + @commitlint/config-conventional（可扩展规则以强制 emoji 出现在标题最前）
    - Husky (git hooks) 或 lefthook 用于在提交时校验格式
    - pre-commit 钩子用于代码格式化与静态检查
- 示例思路：
    - 在 commitlint 配置中添加自定义 rule，检查标题以 emoji 开头，或使用社区插件（例如 commitlint-plugin-emoji）来校验 emoji 与 type 的对应关系。
    - 在 CI 中加入提交规范检查步骤，阻止不合规的合并。

与 PR、Changelog 的协作

- PR 标题建议与相关提交保持一致或更具语义；PR 列表会显示 emoji，方便一眼识别。
- 使用 Conventional Commits 可以与 semantic-release、standard-version 等工具自动生成 CHANGELOG 和语义版本；在生成工具中可忽略 emoji 或将其映射为 type。

覆盖与例外 / Overrides & exceptions

- 项目级覆盖：
    - 项目可在仓库根创建 .repo-commit-instructions.md 或放在 .github/ 目录以覆盖局部细节（例如类型扩展、行宽或 emoji 映射）。
    - 覆盖不得放宽安全/合规相关要求（例如 secrets、PII 的处理）。
- 临时例外须通过 PR 描述和审批记录。

常见问题 / FAQ Q: 提交信息应该是英文还是中文？ A: 参见上文“语言选择”；总体原则为一致性和协作者可读性。公开项目优先英文；内部项目可用中文；也可以采用“标题英文 /
正文中文”的折衷方案。无论语言如何，标题开头必须包含 emoji。 Q: 我忘记写符合规范的提交，如何修正？ A: 使用 git commit --amend 修改最近一次提交；对多个提交可使用 git rebase -i
交互式重写历史（注意团队协作风险）。修正时确保保留或补上正确的 emoji。

示例 PR 检查清单（可放入 PR 模板）

- [ ] 提交信息遵循本规范（emoji + 类型/范围/摘要/必要时包含 BREAKING CHANGE）
- [ ] 本次变更包含相应测试（若适用）
- [ ] 已运行并通过本地/CI linters 与 tests
- [ ] 无硬编码凭证或 PII

维护与负责人

- 维护者：@<team-or-person>
- 变更流程：本文件修改需通过 PR，至少一名拥有写权限的审阅者批准。

下一步建议（可选）

- 在仓库 README 或 CONTRIBUTING 中声明并链接本规范。
- 如果需要我可以为你生成 commitlint + husky 的配置示例，包含一条 rule 强制标题以 emoji 开头，并把这些配置放入一个 PR 中提交到指定仓库。

许可证与法律

- 遵守仓库 LICENSE；提交信息也应避免包含敏感或受限信息。

附录：快速模板（copy/paste）
<emoji> type (scope): short imperative summary

Optional: longer description of the change, motivation and notes.

Footer: Closes #123, BREAKING CHANGE: ...
