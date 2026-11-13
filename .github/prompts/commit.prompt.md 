# Generate Git Commit Message / 生成 Git 提交信息

根据当前变更生成一条高质量的 Git commit message。 Generate a high-quality Git commit message from the current changes.

严格遵循以下主规则（优先级最高）： Follow this primary source of rules (highest priority):

- `.github/instructions/commit.instructions.md`

## 生成要求 / Generation Requirements

1. 先理解本次变更意图（功能、修复、重构、文档、测试、构建或其他）。 First determine the intent of the change (feature, fix, refactor, docs, test, build, or others).
2. 严格按 `commit.instructions.md` 的格式与约束输出。 Follow the exact format and constraints from `commit.instructions.md`.
    - 标题必须以语义一致的 **emoji** 开头。 The title must start with a semantically correct **emoji**.
    - 使用 Conventional Commits 结构：`<emoji> type (scope): summary`。 Use Conventional Commits format: `<emoji> type (scope): summary`.
    - 标题简洁、祈使句、无句号，建议不超过 50 字符。 Keep the title concise, imperative, no trailing period, preferably within 50 characters.
    - 如有必要补充 body（说明 why / 影响 / 风险）。 Add a body when needed (why / impact / risks).
    - 如有必要补充 footer（如 `Closes #123`、`BREAKING CHANGE:`）。 Add a footer when needed (e.g., `Closes #123`, `BREAKING CHANGE:`).
3. 语言策略遵循规则文件：默认标题英文；正文可中文补充，但整体保持一致。 Follow language policy in the rules: title should default to English; body may include Chinese when
   appropriate, while keeping consistency.
4. 若存在破坏性变更，必须明确标注 `BREAKING CHANGE:`。 If there is a breaking change, explicitly include `BREAKING CHANGE:`.
5. 若信息不足以判断类型、范围或风险，先提出澄清问题，不输出模糊提交信息。 If information is insufficient to determine type, scope, or risk, ask clarifying questions instead
   of generating a vague commit message.

## Type 与 Emoji 判定规则 / Type-Emoji Decision Rules

- 必须先判定 `type`，再使用对应 emoji；禁止默认使用 `refactor/♻️`。 You must determine `type` first, then use the matching emoji; never default to `refactor/♻️`.
- 仅当“代码结构重组且不新增功能、不修复缺陷”时，才可使用 `refactor/♻️`。 Use `refactor/♻️` only for structural code changes with no new features and no bug fixes.
- 参考优先级（从高到低）： Suggested priority (high to low):
    1. 修复缺陷 -> `fix/🐛`
       Bug fix -> `fix/🐛`
    2. 新增功能 -> `feat/✨`
       New behavior/feature -> `feat/✨`
    3. 文档变更 -> `docs/📝`
       Documentation-only changes -> `docs/📝`
    4. 测试变更 -> `test/✅`
       Test-only changes -> `test/✅`
    5. 构建或依赖/工具链 -> `build/🛠️` 或 `ci/⚙️`
       Build/dependency/tooling -> `build/🛠️` or `ci/⚙️`
    6. 纯重构 -> `refactor/♻️`
       Pure refactor -> `refactor/♻️`
    7. 其他杂项 -> `chore/🔧`
       Miscellaneous -> `chore/🔧`

## 输出格式 / Output Format

仅输出最终 commit message 内容本身，不要添加解释性文字、前后缀或代码围栏。 Output only the final commit message itself. Do not add explanations, prefixes/suffixes, or code
fences.
