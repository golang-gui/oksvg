# SVG 在线验证与已知限制

日期：2026-10-29。环境：Linux amd64、Go 1.25.8。
首阶段修改 oksvg；后续轮廓修正同时修改自维护 rasterx / scanFT，GOUI 未修改。
解析修正基线：ad6d595。轮廓修正以首阶段的工作区结果为基线；两阶段结果分别记录。
输入 SVG 固定版本、原样读取；不删除属性或在使用方补偿绘制。

## 验证集

| 来源 | 固定提交 | SVG 数 | 许可 |
| --- | --- | ---: | --- |
| [Lucide](https://github.com/lucide-icons/lucide) | be09619c1c1124ad09c19fc3e87ee3b939409bbd | 1,999 | ISC / MIT，见原 LICENSE |
| [Bootstrap Icons](https://github.com/twbs/icons) | e568465481c0c541f63a78f6452d09387711d489 | 2,078 | MIT |
| [resvg-test-suite](https://github.com/linebender/resvg-test-suite) | d8e064337faf01bc5a9579187a56dbdbe3eacc72 | 1,679 | MIT |

全量素材、工具、位图与逐项结果放在仓库外。用于包内回归的十七个原始 SVG 位于
`testdata/corpus/`，附原许可证及 SOURCE.txt；测试不访问网络。

## 已修正

- A/a 的 large-arc / sweep flag 按单字符读取，允许 `a5 5 0 0110 0`，拒绝非法 flag。
- 路径必须以 M/m 开始，不能给缺少 moveto 的非法路径凭空补上起点。
- 数值与长度分开解析；px/in/cm/mm/q/pt/pc 按 96 dpi 换算。
- 几何、描边及虚线的百分比按视口宽、高或归一化对角线计算；不重复加上 viewBox 原点。
- 带显式 viewBox 的 em/ex/rem 根尺寸可作为无需解析字体的固有尺寸提示；形状内的
  字体相对长度仍不猜测。
- HSL 支持 Hue 环绕、小数和角度单位，钳制 S/L；增加 HSLA 及 RGB/HSL 空格、slash 语法。
  逗号形式混用 RGB 整数/百分比仍是非法输入。
- 渐变 pad 保留起点之前的区域；相同起始 offset 不再使左侧延伸取到后一 stop 的颜色。
- 支持本地 paint URL 后的颜色 / currentColor / none fallback；外部资源不会被加载。
- 支持 CSS 注释、静态 a 容器及 SVG 2 椭圆缺失半径的 auto 默认值。
- 圆、椭圆从三点钟位置顺时针构造四段圆弧，修正虚线的上下镜像；没有依赖
  rasterx 对起点等于终点的完整圆弧特殊情况。

修正了两类相关测试输入/预期：

1. 旧 `TestShapes.svg` 的 `rgb(0,100%,255)` 是无效的逗号式混合单位颜色，改为等价的
   `rgb(0,255,255)`，保留描边绘制覆盖；另以负面断言保留非法颜色的检测。
2. 非零 viewBox 下的 userSpaceOnUse 百分比不应加上原点。对应测试中心色由
   RGB(128,0,128) 改为 RGB(64,0,191)，依据独立 Chrome 采样与坐标公式，不放宽容差。

## 运行结果

所有原始输入均尝试 Strict 解析；成功者渲染 24×24、64×64、128×128。

| 验证集 | 修正前完成渲染 | 修正后完成渲染 | 修正后拒绝 |
| --- | ---: | ---: | ---: |
| Lucide | 1,789 | 1,999 | 0 |
| Bootstrap | 2,078 | 2,078 | 0 |
| resvg | 425 | 387 | 1,292 |

这里的“完成渲染”不表示所有像素正确。合计 5,756 个原始 SVG，4,464 个生成 13,392 张
位图；没有解析/绘制 panic，也未发现 RGB 大于 alpha 的非法预乘像素。

resvg 新接受了 28 个用例，涉及紧凑圆弧、百分比长度、函数颜色及 paint fallback。
另有 66 个旧的“渲染成功”现在被明确拒绝：包括未支持的 CSS 选择器、混合模式、
transform-origin、条件处理、fr，以及非法路径/数值等。它们不计作通过，也不宣称全是
渲染器问题。不能通过扩大 Ignore 范围来提高接受数量。

参考：Chrome 156.0.8078.47、librsvg 2.52.5，以及 resvg 套件自带 PNG。
对全部可渲染输入进行 Chrome 对照；对 resvg 的 387 项还按其 PNG 原始尺寸复核。
具体通过/失败/限制归类见下文，不能把二进制能启动或 PNG 能生成当作正确性验收。

## 已修复的底层描边缺陷

原 `rasterx v0.0.0-20220730225603-2ab79fcdd4ef` 在部分反向 round join 处产生缺失轮廓。
已确认的实际图标包括 Lucide braces、dumbbell、squirrel。它们是实现缺陷，后续阶段已在
自维护 rasterx 修复，不再作为 renderer skip 排除通过判定。

控制实验不经过 SVG 解析、SVG 变换或 oksvg 描边适配，直接使用 rasterx/scanFT：

```go
r.SetStroke(12*64, 4*64, rasterx.RoundCap, rasterx.RoundCap,
    nil, rasterx.Round, nil, 0)
r.Start(rasterx.ToFixedP(40, 64))
r.Line(rasterx.ToFixedP(80, 64))
r.Line(rasterx.ToFixedP(40, 64))
r.Stop(false)
r.Draw()
```

在透明 128×128 画布上使用不透明画刷：旧实现中心 (60,64) alpha=255，(84,64) alpha=0。
按半径 6 的圆形转折轮廓，该像素中心距 (80,64) 约 4.53，应完全在内。修复后该处为
不透明画刷颜色。rasterx 的回归覆盖水平、垂直、斜向和近似折返，以及同向和直角控制组，
使用独立的点到线段距离在 1x、2x、4x 验证，而非仅比较两条渲染路径。

另修复共线二次、三次曲线内部的方向反转：解导数驻点后拆分，在几何层补全圆形笔刷扫过的
轮廓，并恢复后续命令的连接配置。合成用例以数学极值点断言内外像素，也与 Chrome 对照。
未使用按图标名称补形状或在 oksvg 使用额外圆点掩盖。

## 后续阶段：轮廓与颜色验证

本阶段新增修正：

- rasterx 的反向 Round 连接，以及共线曲线内部折返。
- oksvg 的 Z 后非 M 命令：从原子路径起点建立新子路径，修复 Lucide mop 的错误填充范围。
- scanFT 的 ColorFunc + draw.Src：修复绿色与蓝色通道交换；Src/Over 与 0、128、255
  覆盖率逐像素对照标准库 DrawMask，包含半透明颜色、背景、非零原点与子图 Stride。
  默认 Over 分支不含该处通道错误，不能把它解释为所有 SVG 曾出现颜色交换。
- oksvg 的 SVG 2 几何属性：基础形状、根尺寸和 use 的 x/y 复用属性/class/inline 的局部
  优先级，修复 style-attribute/non-presentational-attribute 的空白输出。几何不自动继承。

原始素材仍为 5,756 项；增加 4 个合成 SVG 对照，每项 24、64、128px，共比较 13,404 张图。
原始素材渲染/拒绝数量与首阶段一致。无 panic、无非法预乘像素；输入 SHA-256 保持一致。

| 128px Chrome 筛选指标 | 首阶段结果 | 本阶段结果 |
| --- | ---: | ---: |
| Lucide geometry_bad 像素 | 553 | 0 |
| Lucide stable_bad 像素 | 149 | 13 |
| Lucide 候选文件 | 15 | 5 |
| Bootstrap geometry_bad 像素 | 0 | 0 |
| Bootstrap stable_bad 像素 | 376 | 376 |

三个尺寸的 Lucide、Bootstrap 均为 geometry_bad=0，没有新增候选。Lucide 23 个文件、
63 张输出发生变化；braces、dumbbell、squirrel、file-braces、pointer-off、popcorn、
pound-sterling、numbers 等轮廓缺口及 mop 多余填充已消除。另有一些变更发生在圆角边界，
原先即没有被标记为差异。典型修正已进行前后/Chrome 目视对照。

4 个合成 SVG 为 reverse、quadratic-cusp、cubic-cusp、closed-continue；全部 12 张参考图
的 stable_bad、geometry_bad 均为 0。真实原始素材加入包内回归，仍附原许可证与固定提交号。
另外，8 项 CSS 几何回归分别在 24、64、128px 与 Chrome 对照，共 24 张参考图，
stable_bad、geometry_bad 也均为 0；本轮合计比较 13,428 张输出。原始 resvg 几何样式用例
在三个尺寸均消除候选；按本库的 SVG 2 子集处理，不沿用旧 SVG 1.1 套件的空白预期。

该筛选不是完整合规证明，也不是“所有已解析 SVG 均正确”的声明。128px 的 Lucide 残留
5 个候选为 parentheses、rocket、scan-face、sticker、variable，共 13 个 stable_bad 像素；
这些没有超出既定一像素邻域的轮廓缺失，保留为边缘覆盖率差异。Bootstrap 不同尺寸仍有
边缘候选；resvg 的 128px 候选由 19 减为 18，未归因项仍不算通过。

本轮产物位于仓库外 `/tmp/svg-correctness-qsehnwiw/`：
`logs/final-render.jsonl`、`logs/final-comparison.jsonl`、`logs/final-summary.json` 和
`results/before-after.png`。它们是临时诊断产物；长期复现依据为固定素材、包内回归及本文记录。

## 明确未实现的能力

Strict 模式拒绝文本、字体、外部图片、filter、mask、clipPath、pattern、marker、
嵌套 SVG、非 1 整体 opacity 等；本轮不引入离屏场景树或新的文字引擎。

增加识别并明确拒绝非零 fr、mix-blend-mode、非默认 paint-order、transform-origin、
条件处理及非简单 class 的 CSS 选择器。完整 CSS、复杂几何表达式和字体相对形状长度
仍未实现；未知属性继续遵循前向兼容策略，Strict 不是完整 SVG 合规验证器。

## 不能误判的参考差异

- defs / linearGradient 的 display:none：Chrome 与规范及套件 PNG 有差异；没有为了
  匹配 Chrome 的空白结果而禁止引用资源。
- 外圆之外的径向焦点：SVG 1.1 要求移动焦点到边界，SVG 2 允许外部焦点。本库仍沿用
  SVG 1.1 的此项语义；不把版本差异作为本轮已确认算法缺陷，也不声称支持完整 SVG 2。
- 圆角、图像最外行/列仍有抗锯齿与 fixed-point 精度差异。例如 Bootstrap heptagon-fill
  的边界覆盖率差异仍保留记录，不与明显的几何缺失混为一谈。
- 零间隔虚线 n-0：此前将角部差异笼统归于参考差异并不准确。复核原始 500px 套件 PNG，
  其分离端帽与 Chrome 一致；旧 librsvg 的结果不同。该项已确认为 Dasher 顶点处理缺陷并修复。
- 前阶段保留的虚线相位、零长度 dash 和 miter-clip 已在第三阶段复核并修正。
  非共线近退化曲线等剩余差异仍保留为待验收项，不自动宣称已修复或归入 renderer skip。

像素比较采用预乘 sRGB。稳定内部与边界分开：参考 5×5 邻域变化 ≤4/255 的区域检测
超过 4/255 的差异；墨迹检测要求一图 alpha>32，另一图 3×3 邻域 alpha 全部 ≤2。
这只用于筛选候选，不把自动候选数当作缺陷数或一致性通过率。低 alpha 的 PNG
非预乘 RGB 不直接比较，避免量化放大。保留全图差异及未归因项，不扩大容差掩盖。

## 检查

- `go test ./... -count=1`：通过（仓库外源码副本，旧测试会输出 PNG）。
- `go test -race ./... -count=1`：通过。
- parser fuzz 15 秒：通过，最终一轮执行 437,524 次，无 panic。
- `CGO_ENABLED=0 go test -c`：Windows amd64、Linux arm64、macOS arm64 编译通过；
  这是目标编译，不是远端原生运行验收。本库未增加原生依赖。

后续轮廓修正阶段的三库检查：

- rasterx / scanFT / oksvg：分别执行 `go test ./... -count=1`，通过。
- rasterx / scanFT / oksvg：分别执行 `go test -race ./... -count=1`，通过。
- oksvg 最终解析 fuzz 15 秒通过，执行 584,491 次，无 panic 或失败输入。
- 三库分别执行 `CGO_ENABLED=0 go test -c`，Windows amd64、Linux arm64、macOS arm64
  共 9 项编译通过。其他平台未执行原生运行；本轮不需要桌面会话。
- 测试运行于仓库外完整源码副本；快照文件哈希检查用于保证验证代码与交付工作区一致。

## 依赖与后续工作

三个 main 使用 golang-gui 模块路径，已将开发占位替换为实际提交的精确伪版本。
依赖提交尚未推送时，通过仓库外临时 go.work 与本地源码联调；没有将 replace 写入各库
的 go.mod。独立消费前需先推送被引用的提交，正式标签发布另按用户指令执行。

- [x] 本阶段的描边、子路径、颜色缺陷修正，以及 SVG 2 几何样式适配及对照。
- [x] 全量测试与目标编译。
- [x] race、解析 fuzz、最终代码快照与格式检查。
- [x] 将模块迁移、各项修复分别提交，并固定 fork 依赖提交版本。
- [ ] 从 master 创建对应上游 PR 分支，保留上游路径重新验证。
- [ ] 按 rasterx -> scanFT -> oksvg 的顺序推送提交；正式标签另行发布。
- [ ] 后续逐项归因虚线、miter-clip 和非共线退化曲线差异；复杂 CSS 几何表达式等能力另行设计。
- [ ] GOUI 接入，等待用户单独指令。

## 第三阶段：边缘覆盖率与虚线精度

基线：oksvg ca33d96、rasterx 786a0e1、scanFT 22c6bb2。环境同上，GOUI 未修改。
在线输入与参考缓存保持原样，新增 Bootstrap 包内素材附原 MIT 许可及固定提交信息。

### 修正与最小回归

1. 浮点转定点统一最近取整。`M2 2` 后累加 140 次 `v0.1` 应到达 y=16；旧输出在
   128px 目标底部 alpha=224，修复后为 255。坐标与变换、贝塞尔离散点也采用一致规则。
2. 重复采样不携带方向，不能把描边法向量清零。native 的圆端帽内部曾为 alpha=243；
   修复后不透明。Lucide parentheses、rocket、scan-face、sticker、variable 的剩余候选因此消除。
3. Dasher 内部累计完整欧氏距离，修复每个小段截断造成的相位漂移；独立用一段和
   448 个共线小段验证解析相位，包含负 offset 和大 offset。
4. dash 在顶点结束时结束该 dash，即使下一 gap 为零也不跨 dash 生成 miter；新 dash
   起始端帽等待出方向切线。独立 L 形区间覆盖 round/square/butt 与普通 gap、零 gap、点状 dash。
5. scanFT 使用 floor 分解负坐标。四分之一像素交集原输出 alpha=128，应约 64；
   双负坐标的 1/16 像素面积原输出 alpha=64，应约 16。反向绕向、完全在外及包围画布的矩形均验证。
6. miter-clip 以浮点计算裁切平面和两侧交点，最终顶点再量化，并保留自定义 JoinGap。
   3-4-5 法向量的解析交点及真实 gap 中心回归通过。

独立像素采样、颜色空间、预乘规则与筛选阈值均未放宽。覆盖率矩形的 1/255 量化容差
来自既有 12 位覆盖率到 8 位 alpha 的转换，不是为了接受旧错误。

### 对照结果

全量仍为 5,756 份原始 SVG；4,464 份完成三个尺寸渲染，1,292 份按既有 Strict 规则拒绝。
13,392 张原始素材输出加 36 张此前合成/样式回归，共对照 13,428 张图片，未出现 panic、
非法预乘像素或输入 SHA-256 改变。

| Chrome 筛选项 | 上阶段 | 本阶段 |
| --- | ---: | ---: |
| Lucide 128px stable_bad | 13 | 0 |
| Lucide 128px 候选数 | 5 | 0 |
| Bootstrap 24px stable_bad | 7 | 0 |
| Bootstrap 64px stable_bad | 253 | 149 |
| Bootstrap 128px stable_bad | 376 | 112 |
| Bootstrap 128px 候选数 | 36 | 21 |
| resvg 128px 候选数 | 18 | 6 |

Lucide 在 24/64/128px 均为 stable_bad=0、geometry_bad=0；Bootstrap 三个尺寸的
geometry_bad 均为 0。额外 36 张合成/样式回归两项指标均为 0。
原先 12 项虚线用例在 128px 的稳定区域及轮廓筛选全部通过。

另以 resvg 原始 500px PNG 复核，指标使用同一公式：

| 用例 | geometry_bad 修正前→后 | stable_bad 修正前→后 | MAE 修正前→后 |
| --- | --- | --- | --- |
| stroke-dasharray/n-0 | 2,304→0 | 2,116→0 | 0.992550→0.009048 |
| stroke-dasharray/on-a-circle | 714→0 | 276→0 | 0.605903→0.077167 |
| stroke-linejoin/miter-clip | 0→0 | 0→0 | 0.040841→0.021285 |

MAE 为预乘 RGBA 四通道平均绝对差，数值范围 0～255；不将其替代轮廓与颜色断言。

### 仍存在的差异

- 本机 Chrome 156.0.8078.47 的 `CSS.supports('stroke-linejoin','miter-clip')` 为 false，
  路径的计算样式为 miter；它仍被原始筛选列为差异，但该参考不适用于 miter-clip。
  本项使用上述数学回归与套件 PNG，不删除用例、不放宽阈值。
- resvg 其余候选为径向焦点的 SVG 版本差异、display 的参考差异及 line-as-curve-2 的近退化曲线；
  本轮未将它们自动视为修复完成。
- Bootstrap 仍有 21 个 128px 边缘候选，其中 5 个此前未标记：globe-americas-fill、
  hand-thumbs-down、mouse3、person-wheelchair、tornado。新增点全部位于第 0 或 127 行，
  alpha 与参考差 5～11，没有超出既定一像素邻域的轮廓差异。原 Instagram 的边缘覆盖率差异
  在最近取整后有所增大，也保留记录。
- 原因是 Path 仍使用 1/64 SVG 单位的固有精度；在画布边缘放大后可产生小于一像素的覆盖率差异。
  本轮不引入新路径类型、重复缓存或按图标特殊吸附；高精度路径属于后续独立设计。

### 检查与交付状态

- [x] 三库 `go test ./... -count=1`、`go test -race ./... -count=1`。
- [x] Dasher fuzz 15 秒，132,349 次；解析 fuzz 15 秒，564,963 次；无失败。
- [x] 三库 `CGO_ENABLED=0 go test -c`，Windows amd64 / Linux arm64 / macOS arm64，共九项通过。
- [x] 原始输入、独立几何/合成回归、跨库对照及代码快照校验。
- [x] 本轮修复、依赖版本与验证文档分别提交，fork 依赖固定到实际提交的精确版本。
- [ ] 从 master 准备上游 PR 分支；推送、标签和 GOUI 升级另按用户指令执行。

全部执行于 Linux amd64、Go 1.25.8 的仓库外源码副本；跨目标仅编译，不代表原生运行验收。
临时产物：`/tmp/svg-edges-bmcfhzre/logs/full-comparison.jsonl`、`full-summary.json`、
`golden-comparison.json`、`chrome-miter-clip-support.json` 与 `results/before-after.png`。
`dash_fuzz_test.go` 使用 go1.18 构建约束，以保持生产库 Go 1.17 元数据要求。

## 规范依据

- [SVG 路径语法](https://www.w3.org/TR/SVG11/paths.html#PathDataBNF)
- [CSS 绝对单位](https://www.w3.org/TR/css-values-3/#absolute-lengths)
- [CSS HSL](https://www.w3.org/TR/css-color-3/#hsl-color)
- [SVG 2 圆和椭圆等价路径](https://www.w3.org/TR/SVG2/shapes.html#CircleElement)
- [SVG 2 几何属性](https://www.w3.org/TR/SVG2/geometry.html)
- [SVG 2 样式优先级](https://www.w3.org/TR/SVG2/styling.html#PresentationAttributes)
- [SVG 1.1 渐变](https://www.w3.org/TR/SVG11/pservers.html)
- [SVG 2 径向焦点变更](https://www.w3.org/TR/SVG2/pservers.html#RadialGradientNotes)
