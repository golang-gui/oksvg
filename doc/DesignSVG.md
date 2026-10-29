# SVG 解析与栅格化设计

日期：2026-10-29。范围：golang-gui/oksvg；本轮不修改 GOUI，也不升级其依赖。

## 目标与归属

SVG 语义只有一个实现来源。oksvg 负责 XML/命名空间、SVG 样式与继承、路径、变换、
填充规则、渐变引用、viewBox 和 CPU 栅格化。GOUI 后续仅保留资源大小限制、IconSource、
原色/主题色策略与缓存；不得重写 SVG XML 或逐形状修正 SVGPaths。

保持 ReadIconStream、SetTarget、Draw 等现有公共接口，新增：

```go
func (s *SvgIcon) Rasterize(width, height int) *image.RGBA
```

返回独立、透明初始化、预乘 RGBA 位图。非法目标尺寸返回 nil；合法解析结果在合法尺寸
下可栅格化，不修改图标、路径、渐变、dash 或此前返回的图片。不隐藏程序缺陷/OOM。
同一个未被调用者修改的解析结果可并发 Rasterize；每次调用独立创建 scanFT/rasterx 状态。
scanFT 是纯 Go，支持 nonzero/evenodd，不使用忽略 SetWinding 的 ScannerGV 作默认入口。
scanFT 固定版本的 LICENSE 为 BSD-3-Clause；不增加 cgo 或原生运行库依赖。
旧 Draw 仍由调用者提供扫描器，evenodd 是否生效取决于扫描器能力。

Rasterize 按文档 preserveAspectRatio 映射到整张目标图，默认 xMidYMid meet。
SetTarget 保留显式拉伸的既有语义，但修正非零 viewBox 原点的变换顺序。

## 兼容与错误

- DOCTYPE、注释、XML processing instruction 可接受；不加载 DTD，不展开自定义实体，
  不访问网络或外部文件。未定义实体引用由 encoding/xml 报错。
- SVG 根允许 SVG URI 或无命名空间的传统写法。属性按完整命名空间处理，不将
  vendor:fill/vendor:transform 视为 SVG 属性；xml:space 等 XML 元数据不改变路径绘制。
- 外部命名空间元素、SVG metadata 跳过整棵子树，不能泄漏其中的路径到画面。
  title/desc 可保留文字。clip-rule 允许存在，但不能代替 fill-rule。
- 复用 Ignore/Warn/StrictErrorMode。未知属性按前向兼容忽略；已知未实现的绘制能力在
  Strict 模式报错。未知绘制元素在宽松模式跳过子树，不穿透容器绘制其内容。
- XML/数值/路径错误、缺失资源、循环引用等返回错误；不返回成功的部分文档，不用 recover
  掩盖越界。嵌套/引用有明确上限；不建立完整不可信内容沙箱的承诺。

## 编译

XML 读成解析期私有元素结构，一次收集 ID/渐变/class，再编译可见路径；解析结构在返回后
释放，不保留用于绘制的场景树。保留现有路径几何解析器。先收集资源，允许向后引用；
use 解析在当前实例样式/变换下进行，检测递归引用。Defs 公共字段保持兼容。

样式优先级：presentation attributes < 已支持的 class 规则 < inline style，与 XML
属性顺序无关。计算样式归属当前元素：fill-rule、fill/stroke-opacity、color 等继承后
局部覆盖；opacity 不冒充 fill-opacity。默认描边宽度 1，join=miter。
currentColor 在绘制叶子和 stop 中按各自计算 color 解析，不通过文本替换完成。

整体 opacity 需要对象/组离屏合成，当前平面路径表示不能正确完成；本轮不新建绘制场景树，
Strict 模式对尚未实现的非 1 opacity 报错。clip/mask/filter、SVG 文本、外部图片、动画
等仍是明确的后续能力。不能把 GOUI 的整个白名单搬入本库；已有 use/样式/渐变能力保留。

## 绘制

绘制只读取解析数据，每条路径的临时样式、渐变和变换使用局部副本。
填充按完整变换栅格化；描边在局部空间生成轮廓后变换到设备空间，dash 长度保持局部度量。
等比变换可有等价快速路径；非均匀/剪切不能用平均缩放近似。考虑 fixed 点精度，必要时
先放大局部几何再用剩余矩阵变换轮廓。退化变换安全跳过。

渐变以局部几何包围盒计算 objectBoundingBox，使用目标变换、对象变换、bbox 映射与
渐变变换的组合逆矩阵采样；userSpaceOnUse 使用用户空间。几何包围盒不包含描边。
不得因旋转/剪切改用设备轴对齐包围盒，不修改共享 gradient.Stops/Bounds。
渐变百分比按最终继承后的 gradientUnits 计算；userSpaceOnUse 参照 viewBox，
gradientTransform 从单位矩阵解析，不能重复带上定义所在容器的几何变换。
默认 sRGB 中分别插值非预乘颜色和 alpha，最后预乘输出，保留透明 stop 的 RGB。

## 测试与验收

- public 测试集中覆盖 XML 兼容、元数据子树、Strict/Ignore、引用和非法输入；
  白盒测试覆盖继承、填充规则、透明度、变换、数值边界与默认值。
- Rasterize 使用独立几何/数值断言：孔洞、描边宽度、虚线、透明合成、非零原点、
  preserveAspectRatio、渐变。对复杂素材与固定版本 librsvg 作对照，不能只比较两个
  可能同错的实现。抗锯齿边界与内部像素分开检查，不扩大容差掩盖错误。
- 同一图标并发渲染多尺寸，像素结果稳定，race 通过；旧 SetTarget/Draw 用例保留。
- 非法输入 fuzz 必须无 panic；有可解释的根因修正，不在 GOUI 恢复兜底。
- 旧测试会写 testdata PNG，完整测试在仓库外源码副本执行；新测试只用内存/t.TempDir。
- 执行 go test ./...、race、fuzz 和 Windows amd64/Linux arm64/Darwin arm64 编译。
  性能只在同资源、同尺寸、同栅格器下测量；不因移除 GOUI 重复编译就承诺未经测量的收益。

## 实施阶段

- [x] XML、命名空间、资源收集与兼容解析。
- [x] 样式、填充、透明度覆盖、currentColor 与默认值。
- [x] 描边/渐变变换、viewBox 和 Rasterize。
- [x] 真实资源、全量/race/fuzz、跨目标验证，更新能力说明。
- [ ] 后续由用户单独指示 GOUI 依赖升级及删除补偿；本轮不执行。

## 本轮验证记录

环境：Linux amd64、Go 1.25.8；纯 Go CPU 栅格化，无桌面或显示连接要求。
完整测试在仓库外源码副本运行，避免旧用例将 PNG 写入工作区。新增测试不输出源码目录文件。

- 基线 75e7ce8 全量测试通过。
- 新实现 `go test ./... -count=1` 与 `go test -race ./... -count=1` 均通过。
- `go test -run '^$' -fuzz '^FuzzReadIconStream$' -fuzztime=15s -parallel=2`
  通过，执行 475,615 次；没有解析 panic 或错误时返回部分图标。
- `CGO_ENABLED=0 go test -c`：Windows amd64、Linux arm64、macOS arm64 均通过。
  这是目标编译验证，不代表远端原生运行验收；本轮不修改平台接入代码。
- 回归覆盖 DOCTYPE、命名空间、clip-rule、元数据子树、填充规则、透明度覆盖、
  currentColor、class 顺序、向后引用、描边/虚线变换、渐变、viewBox、并发隔离与深度上限。
- 既有 badelement.svg 是不完整的 cubic path；旧测试只保存部分图像。本轮保留素材，
  调整为三种 ErrorMode 均断言 nil/error，不再把畸形几何当作成功的部分绘制。
- 深层 gradient href 的限制曾受 map 遍历和缓存顺序影响；改为保存完整引用深度，
  `TestDocumentLimits -count=100` 验证结果稳定。

独立绘制对照在 160×160 物理像素、默认 sRGB 下进行：

- GOUI 现有 brainSvg、peopleSvg、devToolSvg 只读取原始素材，没有修改 GOUI。
  与 librsvg 2.52.5 及 Chrome 156.0.8078.47 比较，稳定不透明内部 RGB 相同，
  距边界至少一像素的区域未发现几何偏移。
- 线性/径向渐变、渐变旋转、剪切与非均匀描边使用合成 SVG 加独立数学断言。
  Chrome 对照的稳定不透明内部 RGB 最大差异 1/255；抗锯齿边界不当作统一金图。
  浏览器对照显式指定 SVG 视口尺寸，避免 canvas 拉伸 SVG 固有尺寸造成伪差异。
- librsvg 2.52.5 在剪切后的 objectBoundingBox 渐变与坐标公式、Chrome 存在差异；
  该项采用坐标公式及 Chrome 交叉核对，没有通过改变正确的坐标映射迁就旧参考结果。
- 透明渐变用预乘像素独立断言；不直接比较低 alpha PNG 的非预乘 RGB 放大量化误差。

剩余范围：整体 opacity 离屏合成、clip/mask/filter、文本、嵌套 SVG 等未实现的
绘制能力在 Strict 模式明确报错；复杂 CSS 不在支持范围。
能力列表同步更新于 SVG_Element_List.txt。
GOUI 当前 svg.Parse 的前置限制和原有补偿尚未移除，等待用户单独指示。

## 依据

核对基线：75e7ce872d46，本地 fork 与 GOUI 固定版本一致。

- SVG 1.1 painting：https://www.w3.org/TR/SVG11/painting.html
- SVG 1.1 coordinates：https://www.w3.org/TR/SVG11/coords.html
- SVG 1.1 foreign namespaces：https://www.w3.org/TR/SVG11/extend.html#ForeignNamespaces
- SVG 1.1 group opacity：https://www.w3.org/TR/SVG11/masking.html#ObjectAndGroupOpacityProperties
- Qt 6 QSvgRenderer：https://doc.qt.io/qt-6/qsvgrenderer.html
- librsvg render_document（自 2.46）：https://gnome.pages.gitlab.gnome.org/librsvg/Rsvg-2.0/method.Handle.render_document.html

参考用于确认标准语义与资源边界，不引入 QObject、GDK 或 GOUI 依赖。
