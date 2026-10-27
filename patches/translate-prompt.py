#!/usr/bin/env python3
"""本地补丁：生图工作台「中→英润色」按钮。

调文本模型把输入框里的中文描述改写成英文生图提示词后填回输入框，并支持一键还原。
幂等：已打过则跳过（每个改动都有独立的标记判断）。
用法：python3 patches/translate-prompt.py  （apply.py 会自动调用）
"""
import pathlib

path = pathlib.Path(__file__).resolve().parent.parent / "web/src/app/(user)/image/page.tsx"
s = path.read_text(encoding="utf-8")
done = []
skipped = []


def rep(marker, old, new, tag, count=1):
    """marker 已存在则跳过；否则替换 old→new。"""
    global s
    if marker in s:
        skipped.append(tag)
        return
    if old in s:
        s = s.replace(old, new, count)
        done.append(tag)
    else:
        print(f"!! 锚点未命中：{tag}")


# ---------- 1) imports ----------
rep("    Languages,\n", "    ImagePlus,\n", "    ImagePlus,\n    Languages,\n", "图标 Languages")
rep("    Undo2,\n", "    Trash2,\n    Upload,", "    Trash2,\n    Undo2,\n    Upload,", "图标 Undo2")
rep(
    "requestImageQuestion",
    'import { ImageRequestError, batchCanvasImageTaskStatus, createCanvasImageTask, deleteCanvasImageTask, listCanvasImageTasks, requestEdit, requestGeneration, type CanvasImageTask } from "@/services/api/image";',
    'import { ImageRequestError, batchCanvasImageTaskStatus, createCanvasImageTask, deleteCanvasImageTask, listCanvasImageTasks, requestEdit, requestGeneration, requestImageQuestion, type CanvasImageTask } from "@/services/api/image";',
    "引入 requestImageQuestion",
)

# ---------- 2) 模块级系统提示词 ----------
rep(
    "TRANSLATE_PROMPT_STRIP",
    'const WORKFLOW_BUTTON_POSITION_KEY = "infinite-canvas:workflow-button-position";',
    'const WORKFLOW_BUTTON_POSITION_KEY = "infinite-canvas:workflow-button-position";\n'
    "// 本地补丁：中文提示词 → 英文生图提示词（交给文本模型润色）\n"
    "const TRANSLATE_PROMPT_SYSTEM = [\n"
    '    "你是资深 AI 绘图提示词工程师。把用户的中文（或中英混合）描述改写成一整段高质量英文生图提示词。",\n'
    '    "要求：",\n'
    '    "1) 只输出英文提示词本身，不要解释、不要加引号、不要 Markdown、不要换行；",\n'
    '    "2) 完整保留原意与所有具体细节，补充镜头、光线、材质、色彩、画幅等通用摄影或插画术语，但不要凭空增加原文没有的主体、品牌或文字；",\n'
    '    "3) 原文已含英文时保留英文并润色；",\n'
    '    "4) 输出为逗号分隔的自然短语，60 词以内；",\n'
    '    "5) 不要输出任何中文。",\n'
    '].join("\\n");\n'
    "const TRANSLATE_PROMPT_STRIP = /^\\s*(```[a-zA-Z]*|英文提示词[:：]?|English prompt[:：]?)\\s*|\\s*```\\s*$/g;",
    "系统提示词常量",
)

# ---------- 3) 父组件：状态 + 函数 ----------
rep(
    "translatingPrompt",
    '    const [prompt, setPrompt] = useState("");',
    '    const [prompt, setPrompt] = useState("");\n'
    "    const [translatingPrompt, setTranslatingPrompt] = useState(false);\n"
    '    const [promptBeforeTranslate, setPromptBeforeTranslate] = useState("");',
    "父组件状态",
)

rep(
    "const translatePrompt",
    """    const clearPrompt = () => {
        setPrompt("");
    };
""",
    """    const clearPrompt = () => {
        setPrompt("");
    };

    // 本地补丁：中→英润色（调文本模型，把中文想法改写成英文生图提示词后填回输入框）
    const translatePrompt = async () => {
        const source = prompt.trim();
        if (!source) {
            message.warning("先在提示词框里写下中文想法");
            return;
        }
        const promptModel = (effectiveConfig.textModel || "").trim();
        if (!promptModel) {
            message.warning("请先在「配置与用户偏好 → 默认文本模型」里选一个文本模型（如 gpt-5.5），并确保这个渠道的模型列表里有它");
            openConfigDialog(true);
            return;
        }
        const textConfig: AiConfig = {
            ...effectiveConfig,
            model: promptModel,
            textModel: promptModel,
            textChannelId: effectiveConfig.textChannelId,
            activeChannelId: effectiveConfig.textChannelId,
        };
        if (!isAiConfigReady(textConfig, promptModel)) {
            message.warning("请先配置文本模型（配置与用户偏好 → 默认文本模型）");
            openConfigDialog(true);
            return;
        }
        setTranslatingPrompt(true);
        try {
            const answer = await requestImageQuestion(textConfig, [
                { role: "system", content: TRANSLATE_PROMPT_SYSTEM },
                { role: "user", content: source },
            ], () => {});
            const cleaned = answer.replace(TRANSLATE_PROMPT_STRIP, "").trim();
            if (!cleaned) {
                message.warning("模型没有返回内容，换个文本模型再试");
                return;
            }
            setPromptBeforeTranslate(source);
            setPrompt(cleaned);
            message.success("已改写成英文提示词，可直接点开始生成");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "润色失败，请检查文本模型配置");
        } finally {
            setTranslatingPrompt(false);
        }
    };

    const undoTranslatePrompt = () => {
        if (!promptBeforeTranslate) return;
        setPrompt(promptBeforeTranslate);
        setPromptBeforeTranslate("");
        message.success("已还原为原来的中文提示词");
    };
""",
    "父组件 translatePrompt",
)

# ---------- 4) 子组件：props ----------
rep(
    "canUndoTranslate,",
    """    onPastePrompt,
    onClearPrompt,
    onPasteReferences,
    onUploadReferences,
    onRemoveReference,
    onGenerate,
    uploadingCount,
}: {
    layout: WorkbenchLayout;""",
    """    onPastePrompt,
    onClearPrompt,
    onPasteReferences,
    onUploadReferences,
    onRemoveReference,
    onGenerate,
    uploadingCount,
    translatingPrompt,
    canUndoTranslate,
    onTranslatePrompt,
    onUndoTranslate,
}: {
    layout: WorkbenchLayout;""",
    "子组件解构",
)

rep(
    "translatingPrompt: boolean;",
    """    onClearPrompt: () => void;
    onPasteReferences: () => void;""",
    """    onClearPrompt: () => void;
    translatingPrompt: boolean;
    canUndoTranslate: boolean;
    onTranslatePrompt: () => void;
    onUndoTranslate: () => void;
    onPasteReferences: () => void;""",
    "子组件 props 类型",
)

# ---------- 5) 两个调用点传参 ----------
rep(
    "translatingPrompt={translatingPrompt}",
    """                            onPastePrompt={() => void pastePromptFromClipboard()}
                            onClearPrompt={clearPrompt}
                            onPasteReferences={() => void addReferencesFromClipboard()}""",
    """                            onPastePrompt={() => void pastePromptFromClipboard()}
                            onClearPrompt={clearPrompt}
                            translatingPrompt={translatingPrompt}
                            canUndoTranslate={Boolean(promptBeforeTranslate)}
                            onTranslatePrompt={() => void translatePrompt()}
                            onUndoTranslate={undoTranslatePrompt}
                            onPasteReferences={() => void addReferencesFromClipboard()}""",
    "调用点传参 x2",
)

# ---------- 6) 底部悬浮布局按钮 ----------
rep(
    "中文转英文提示词",
    """                                <Button title="清空输入" icon={<Trash2 className="size-4" />} onClick={onClearPrompt} />
                                <Button title="提示词库" icon={<BookOpen className="size-4" />} onClick={onOpenPromptLibrary} />""",
    """                                <Button title="清空输入" icon={<Trash2 className="size-4" />} onClick={onClearPrompt} />
                                <Button title="中文转英文提示词（调文本模型润色）" loading={translatingPrompt} icon={<Languages className="size-4" />} onClick={onTranslatePrompt} />
                                <Button title="还原成润色前的中文" disabled={!canUndoTranslate} icon={<Undo2 className="size-4" />} onClick={onUndoTranslate} />
                                <Button title="提示词库" icon={<BookOpen className="size-4" />} onClick={onOpenPromptLibrary} />""",
    "底部布局按钮",
)

# ---------- 7) 侧边工作台按钮 ----------
rep(
    "中→英润色",
    """                            <Button size="small" icon={<ClipboardPaste className="size-3.5" />} onClick={onPastePrompt}>读取剪贴板</Button>
                            <Button size="small" icon={<Trash2 className="size-3.5" />} onClick={onClearPrompt}>清空</Button>
                            <Button size="small" icon={<BookOpen className="size-3.5" />} onClick={onOpenPromptLibrary}>提示词库</Button>""",
    """                            <Button size="small" icon={<ClipboardPaste className="size-3.5" />} onClick={onPastePrompt}>读取剪贴板</Button>
                            <Button size="small" type="primary" ghost loading={translatingPrompt} icon={<Languages className="size-3.5" />} onClick={onTranslatePrompt}>中→英润色</Button>
                            <Button size="small" disabled={!canUndoTranslate} icon={<Undo2 className="size-3.5" />} onClick={onUndoTranslate}>还原</Button>
                            <Button size="small" icon={<Trash2 className="size-3.5" />} onClick={onClearPrompt}>清空</Button>
                            <Button size="small" icon={<BookOpen className="size-3.5" />} onClick={onOpenPromptLibrary}>提示词库</Button>""",
    "侧边布局按钮",
)

path.write_text(s, encoding="utf-8")
print("本次改写:", done or "（无）")
print("已是目标状态:", skipped or "（无）")
chk = path.read_text(encoding="utf-8")
print("translatePrompt 定义数:", chk.count("const translatePrompt"))
print("onTranslatePrompt 引用数:", chk.count("onTranslatePrompt"))
print("无占位符残留:", "***" not in chk)
if done:
    print("提示：改动已写入，需要 ./start.sh rebuild 才生效")