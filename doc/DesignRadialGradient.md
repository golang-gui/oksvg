# SVG 2 径向渐变与参考判定

## 范围与现状

本阶段保持 GOUI 不变，不修改输入 SVG。SVG 的填充采样位于 oksvg `paint.go`；
rasterx 的旧 `Gradient.GetColorFunction` 并非此绘制路径，本阶段不迁移该旧接口。

原 SVG 采样把圆外焦点钳到圆内，沿用 SVG 1.1；SVG 2 允许圆外焦点，
渐变覆盖的圆锥之外应保持透明。相同输入可能产生大面积外观差异，不能归为抗锯齿误差。
目前仍仅支持零焦点半径 `fr=0`，非零 fr 的 Strict 拒绝规则不在本轮扩大。

## 参数求解

在已有组合逆变换后的渐变坐标内，令 F 为焦点、C 为终圆圆心、r 为终圆半径，
Q=P-F、D=C-F。参数 t 对应圆心 F+tD、半径 tr，满足：

```text
(D·D-r²)t² - 2(Q·D)t + Q·Q = 0
```

- 取最大的非负实根；不存在有效圆时透明，不能填充最后一 stop。
- 圆内焦点保留原有圆形渐变语义；圆外焦点不再钳制。
- 焦点恰在终圆上时，退化为一次方程；避免除零和不稳定的相减。
- 再按既有 pad / repeat / reflect 映射 t，复用透明 stop、sRGB 插值和预乘输出。
- r=0 仍按退化渐变填充最后一 stop；保留现有变换和共享资源只读契约。

## 独立验证

使用圆心和半径已知的 t 等值圆选取像素中心，直接给出颜色预期；
覆盖内/外/边界焦点、无实根、所有根在负方向、重复/反射、透明 stop 与变换。
不把另一条渲染路径或 Chrome 输出作为唯一断言。
原始 focal-point-correction.svg 同时对照 Chrome 和 resvg 套件 PNG。

对照结果必须分为：实际实现缺陷、已修复、参考不适用、未支持和普通边缘差异。
`display:none` 的 defs / linearGradient 仍可被引用；根 svg 的 display:none 应不绘制。
这些用例以规范与套件预期验证，不为匹配本机 Chrome 的相反输出改变正确实现。
Chrome 不支持 miter-clip 时，该属性仍以数学几何和套件 PNG 验收。

## 状态

- [x] 零焦点半径的 SVG 2 圆外/边界焦点求解及独立回归。
- [x] 原始 SVG、全量图标、race、fuzz 和无 cgo 目标编译验证。
- [x] 更新带明确参考分类的前后对照图与 CorpusValidation。

SVG 2 的非零 fr、边界焦点配合 repeat/reflect 的退化区域平均色等完整圆族边界语义
不在本阶段的支持承诺内，不能将本轮圆外焦点修正描述为完整 SVG 2 渐变实现。

## 依据

- [SVG 2 径向渐变](https://www.w3.org/TR/SVG2/pservers.html#RadialGradientNotes)
- [SVG 1.1 的旧焦点钳制规则](https://www.w3.org/TR/SVG11/pservers.html#RadialGradients)
- [SVG 2 paint servers 与 display](https://www.w3.org/TR/SVG2/pservers.html#Introduction)
- [SVG 2 显示规则](https://www.w3.org/TR/SVG2/render.html#VisibilityControl)
