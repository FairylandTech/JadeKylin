# Global Copilot Instructions

说明

- 目的：为 Copilot / AI 代码助手（以及团队成员）提供一份可读、可复用的全局指令集，明确代码风格、约束、优先级和不可触碰的安全边界。
- 适用范围：适用于本仓库中由 AI 生成或由 AI 协助的代码、提交信息、注释与文档。各子项目可在本文件之上增加或覆盖项目级指令（见“仓库/项目覆盖”）。

版本与维护

- 版本：v1.0
- 最后更新：2025-09-04
- 维护者：@<team-or-person>
- 更新流程：对本文件的更改需通过 PR，至少一名拥有写权限的审阅者批准；涉及安全或合规条目的修改需额外得到安全负责人批准。

基本原则（优先级从高到低）

1. 安全与合规（最高优先）
    - 不生成或引入任何硬编码的秘密（API 密钥、密码、私钥等）。
    - 不生成有意规避法律或违反第三方许可的代码。
    - 遵守仓库中已有的安全策略（SCA、依赖白名单等）。
2. 正确性与可维护性
    - 优先输出易于理解、容易测试且有良好边界条件处理的实现。
    - 避免复杂的单行魔法式实现；必要时拆成小函数并添加单元测试。
3. 可读性与风格一致性
    - 遵守仓库的代码风格（见“代码风格”一节）。若仓库中存在 linter/formatter 配置（例如 .prettierrc、.eslintrc、.editorconfig、pyproject.toml），以其为准。
4. 性能考虑（按需）
    - 在明显会导致性能问题的场景下，优先使用更高效的算法；否则以可读性优先。
5. 最小权限原则（对权限和网络访问）
    - 不添加额外运行时权限或网络调用，除非有明确需求与安全审查通过。

代码风格（示例）

- 语言：按文件类型检测（Python、TypeScript、Java、Go 等）。
- PyCharm（仓库默认 code style 指南）
    - 本仓库在 IDE 级别推荐使用 PyCharm Code Style 作为团队共享设置，主要约束：
        - 右边界（Right margin / line length）设为 170。
        - 打开自动换行与常用格式化选项以便与团队一致。
        - 将项目的 Code Style 导出为 XML 并放入仓库（例如 .idea/ 或 .github/ 或 docs/ 下的 code-style 目录），以便团队导入使用。
    - 可导入的 PyCharm Code Style XML 示例（只示范设置 RIGHT_MARGIN = 170）：

```xml
<?xml version="1.0" encoding="UTF-8"?>
<code_scheme name="Project" version="173">
    <!-- 右边界（行宽） -->
    <option name="RIGHT_MARGIN" value="170"/>
    <!-- 其他建议设置（可按需调整） -->
    <option name="WRAP_ON_TYPING" value="true"/>
    <option name="CONTINUATION_INDENT_SIZE" value="8"/>
    <!-- 语言特定设置可在导出文件中包含 -->
</code_scheme>
```

- 编辑器/格式化工具同步建议：
    - 若项目使用 Black（Python），在 pyproject.toml 中设置：

```toml
[tool.black]
line-length = 170
```

    - 可同时在仓库添加 .editorconfig（供多编辑器参考）：

```ini
# EditorConfig for project
root = true

[*]
charset = utf-8
trim_trailing_whitespace = true
insert_final_newline = true
max_line_length = 170
```

- 注意：PyCharm 的 RIGHT_MARGIN 是 IDE 可视/格式化参考值；请同时在 CI/formatter（如 black/prettier）中设置一致的行长以保证一致性。
- Python
    - 以 PEP8/black 为准，但将 line-length 覆盖为 170（见上文）。
    - 使用类型注解（至少公共函数）。
    - 异常处理：不要用 bare except；明确捕获异常类型并记录上下文。
- JavaScript/TypeScript
    - 使用严格模式、适当的类型注解（TypeScript）。
    - 遵守现有 ESLint/Prettier 配置（若仓库没有，建议将 max line width 设置为 170）。
- Commit message
    - 使用简洁的动词现在时：如 "Add userModel authentication", "Fix race condition in cache"。
    - 若仓库使用 Conventional Commits，请遵循该规范。

安全与隐私（具体规则）

- 禁止将任何凭证、秘密、个人身份信息（PII）写入仓库。
- 任何需要外部服务或第三方库的新增依赖，需在 PR 描述中说明用途和风险，并通过依赖审查流程。
- 若实现会处理用户可控输入，必须验证并防护常见攻击（SQL 注入、XSS、命令注入等）。

测试与质量保障

- 新增逻辑必须包含单元测试（或集成测试，视场景而定）。
- 测试应覆盖主要成功路径和关键错误路径。
- CI 失败（lint/test）禁止合并。

文档与注释

- 公共 API、复杂逻辑或非直观实现必须有文档注释（docstring / JSDoc）。
- README、架构图或使用说明应在用户可见的重要变更时更新。

不可做的事情（禁止列表）

- 生成含有真实凭证或秘密的代码。
- 生成可能用于规避安全或监控的工具和脚本（例如：rootkit、后门、远程接入脚本等）。
- 自动提交超大二进制文件或引入未审查的闭源依赖。
- 在未经授权的情况下向外部服务暴露内部数据。

对 AI 的具体行为指示（示例）

- 在修改代码时，写明修改动机、改动范围、相关风险和回滚方法。
- 对于有多种实现方案的任务，先提供两到三种可选方案（包含优缺点与复杂度），然后实现用户选择的方案。
- 如果不确定需求或存在模糊点，应提出 clarifying questions（澄清性问题）而不是做危险假设。
- 在生成大段逻辑或关键变更时，同时生成对应的测试用例草案与必要的文档改动。

仓库/项目覆盖（如何覆盖本文件）

- 项目可以在其根目录创建 .copilot-instructions.md 或在 .github/ 下创建 repo-copilot-instructions.md 来补充或覆盖本全局指令。
- 覆盖规则：项目级指令仅可放宽非安全相关条款（例如代码格式、行宽），但不得覆盖安全/合规/秘密管理相关条款。
- 若项目和全局文件冲突，安全相关条款以全局为准；可通过 PR 请求例外，并注明风险评估与审批人。

示例模板（用于快速开始）

- 简短版（用于自动化提示）
    - "在本仓库中，优先保证安全与可维护性。遵循仓库现有 linter/formatter。新增代码需带测试。禁止将凭证写入代码。遇到模糊需求请提问。"
- 完整版（供 README 或贡献指南链接）
    - 将本文件内容作为贡献者指南的一部分，链接到仓库 README，并在 PR 模板中加入自动核对项（lint、测试、依赖审查、安全检查）。

PR 模板建议（核对清单）

- [ ] 修改目的与背景已描述
- [ ] 新增逻辑包含单元测试
- [ ] 已运行并通过本地/CI lint 与 test
- [ ] 相关依赖已声明并通过审查
- [ ] 无硬编码秘密或 PII

FAQ（常见问题）
Q: 如果 AI 生成了秘密或凭证怎么办？
A: 立即删除该提交，撤回并使用 git rebase / filter-repo 清除历史，通报安全负责人，并在仓库中旋转相关密钥。

Q: AI 可以自动提交 PR 吗？
A: 可以在有审查流程与人类最终批准的前提下；任何自动化 PR 都要包含清晰的变更说明与测试。

联系方式与审批

- 安全负责人：@<security-owner>
- 代码风格/架构联系人：@<arch-owner>

附录：示例 prompts（供团队使用）

- "生成一个遵守本仓库风格的 Python 函数，用于将 ISO8601 字符串解析为带时区的 datetime 对象；包含类型注解与单元测试。"
- "重构模块 X，将超过 200 行的函数拆分为更小的函数，并保持行为不变；提供变更说明与测试更新。"

许可证与法律

- 请遵守本仓库 LICENSE 文件中的条款；AI 生成内容同样受 LICENSE 约束，必要时在 PR 中声明生成方式以便审计。

结束语

- 本文件旨在减少 AI 协作带来的风险并提高产出质量。建议每季度或在重大流程变更时复审本指令。
