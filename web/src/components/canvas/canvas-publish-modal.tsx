"use client";

import { useEffect, useState } from "react";
import { App, Input, Modal, Select } from "antd";

import { pickShowcaseMedia, publishShowcaseWork, SHOWCASE_CATEGORIES } from "@/services/api/showcase";
import type { CanvasProject } from "@/app/(user)/canvas/stores/use-canvas-store";
import { useUserStore } from "@/stores/use-user-store";

// CanvasPublishModal 把画布项目发布到创作精选。
export function CanvasPublishModal({ project, onClose }: { project: CanvasProject | null; onClose: () => void }) {
    const { message } = App.useApp();
    const [title, setTitle] = useState("");
    const [category, setCategory] = useState(SHOWCASE_CATEGORIES[0]);
    const [description, setDescription] = useState("");
    const [publishing, setPublishing] = useState(false);

    const open = Boolean(project);
    // 每次打开时重置为项目当前值，避免上一个项目的表单残留。
    useEffect(() => {
        if (project) {
            setTitle(project.title || "");
            setCategory(SHOWCASE_CATEGORIES[0]);
            setDescription("");
        }
    }, [project]);

    const handlePublish = async () => {
        if (!project) return;
        if (!useUserStore.getState().token) {
            message.warning("请先登录后再发布");
            return;
        }
        const media = pickShowcaseMedia(project);
        if (!media.cover && !media.preview) {
            message.warning("画布里还没有可展示的图片或视频节点");
            return;
        }
        setPublishing(true);
        try {
            await publishShowcaseWork({
                projectId: project.id,
                title: title.trim() || project.title,
                description: description.trim(),
                category,
                cover: media.cover,
                preview: media.preview,
                data: project,
            });
            message.success("已发布到创作精选");
            setTitle("");
            setDescription("");
            onClose();
        } catch (error) {
            message.error(error instanceof Error ? error.message : "发布失败");
        } finally {
            setPublishing(false);
        }
    };

    return (
        <Modal
            title="发布到创作精选"
            open={open}
            confirmLoading={publishing}
            onOk={() => void handlePublish()}
            onCancel={onClose}
            okText="发布"
            cancelText="取消"
            destroyOnHidden
        >
            <p className="mb-4 text-xs text-stone-500">发布后其他用户可以预览作品并查看画布创作过程，可随时在管理后台下架。</p>
            <div className="flex flex-col gap-3">
                <div>
                    <label className="mb-1 block text-[13px] font-medium">作品标题</label>
                    <Input value={title} placeholder="给作品起个名字" maxLength={60} onChange={(event) => setTitle(event.target.value)} />
                </div>
                <div>
                    <label className="mb-1 block text-[13px] font-medium">分类</label>
                    <Select value={category} className="w-full" options={SHOWCASE_CATEGORIES.map((item) => ({ value: item, label: item }))} onChange={setCategory} />
                </div>
                <div>
                    <label className="mb-1 block text-[13px] font-medium">作品简介（可选）</label>
                    <Input.TextArea value={description} placeholder="简单介绍这个作品" rows={3} maxLength={200} showCount onChange={(event) => setDescription(event.target.value)} />
                </div>
            </div>
        </Modal>
    );
}
