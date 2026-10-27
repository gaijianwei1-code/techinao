/* ══════════ Mock 数据（唯一数据源，未来由后端 /api/v1/showcase 提供） ══════════ */

const DB = {
  categories: ["全部", "精选画布", "专业影视", "短剧漫剧", "动漫游戏", "教育生活"],

  works: [
    {
      id: "w-zhihuiyi",
      title: "致回忆",
      author: "编导傅年",
      avatarHue: 12,
      likes: 37, liked: false, followed: false, views: 12840,
      category: "专业影视",
      cover: "assets/cover-zhihuiyi.jpg",
      video: "assets/video-bbb.mp4",
      duration: "2:32", resolution: "3840 × 2160",
      publishedAt: "10月23日",
      desc: "两个少年在村子里的最后一次对视。\n全片用无限画布完成：先写脚本、出角色三视图和分镜，再逐镜生成视频片段。",
      promptTags: ["电影感", "黄昏逆光", "儿童奇幻", "情绪对峙"],
      canvas: {
        nodes: [
          { id: "n1", type: "text", group: "10月22日", x: 0, y: 80, w: 260, h: 150, title: "创意脚本", text: "傍晚的石板路上，蓝发少年与火焰少年对峙。\n风吹起尘埃，夕阳在两人之间落下。\n台词：「你就是故意的。」" },
          { id: "n2", type: "image", group: "10月22日", x: 300, y: 80, w: 300, h: 300, title: "角色三视图", src: "assets/proc-charsheet.jpg", prompt: "主角设定：蓝发少年，正/侧/背三视图" },
          { id: "n3", type: "image", group: "10月22日", x: 640, y: 80, w: 380, h: 285, title: "分镜草稿", src: "assets/proc-storyboard.jpg", prompt: "追逐戏分镜，pan right / tilt down" },
          { id: "n4", type: "image", group: "10月22日", x: 640, y: 405, w: 380, h: 214, title: "场景参考", src: "assets/proc-location.jpg", prompt: "清晨石板巷，雾气 + 暖色灯笼" },
          { id: "n5", type: "config", group: "10月23日", x: 0, y: 420, w: 260, h: 130, title: "出图参数", text: "3840 × 2160 · 8s\n电影级色彩\n参考节点：n2 n4" },
          { id: "n6", type: "video", group: "10月23日", x: 300, y: 480, w: 480, h: 270, title: "正片 · 对峙镜头", src: "assets/video-bbb.mp4", poster: "assets/cover-zhihuiyi.jpg", resolution: "3840 × 2160" },
          { id: "n7", type: "image", group: "10月23日", x: 820, y: 480, w: 360, h: 270, title: "色彩参考", src: "assets/proc-colorscript.jpg", prompt: "暖色日落色板" }
        ],
        edges: [
          { from: "n1", to: "n3" },
          { from: "n2", to: "n5" },
          { from: "n4", to: "n5" },
          { from: "n5", to: "n6" },
          { from: "n6", to: "n7" }
        ]
      }
    },
    {
      id: "w-fluffy",
      title: "毛绒起源丨原来在另一个平行世界，我们都是毛绒",
      author: "小眠",
      avatarHue: 330,
      likes: 12, liked: false, followed: false, views: 4021,
      category: "精选画布",
      cover: "assets/cover-fluffy.jpg",
      video: "assets/video-flower.mp4",
      duration: "0:45", resolution: "1920 × 1080",
      publishedAt: "10月20日",
      desc: "如果世界上所有生物都是毛绒做的会怎样？\n三只毛绒小兽的云上漫步。",
      promptTags: ["毛绒质感", "梦幻", "3D 渲染", "粉色系"],
      canvas: null
    },
    {
      id: "w-battle",
      title: "青雉VS赤犬",
      author: "西瓜君",
      avatarHue: 205,
      likes: 84, liked: false, followed: false, views: 21500,
      category: "动漫游戏",
      cover: "assets/cover-battle.jpg",
      video: "assets/video-bbb.mp4",
      duration: "1:10", resolution: "2560 × 1440",
      publishedAt: "10月18日",
      desc: "冰与岩浆的港湾决战，一镜到底尝试。",
      promptTags: ["战斗", "冰火对决", "高对比", "动漫"],
      canvas: null
    },
    {
      id: "w-sakura",
      title: "《和樱花妹互换身体の日常》第六集（原创ai短剧）",
      author: "我不知豆啊",
      avatarHue: 42,
      likes: 478, liked: false, followed: false, views: 98000,
      category: "短剧漫剧",
      cover: "assets/cover-sakura.jpg",
      video: "assets/video-flower.mp4",
      duration: "3:05", resolution: "1920 × 1080",
      publishedAt: "10月16日",
      desc: "第六集：樱花树下的约定。系列短剧更新中。",
      promptTags: ["日常", "樱花", "治愈", "连续剧"],
      canvas: null
    },
    {
      id: "w-kpop",
      title: "韩国女团K-POP",
      author: "文静0731",
      avatarHue: 280,
      likes: 49, liked: false, followed: false, views: 15600,
      category: "专业影视",
      cover: "assets/cover-kpop.jpg",
      video: "assets/video-bbb.mp4",
      duration: "0:58", resolution: "1920 × 1080",
      publishedAt: "10月14日",
      desc: "虚拟女团 LUMINA 首支舞台，霓虹灯 + 干冰氛围。",
      promptTags: ["舞台", "霓虹", "女团", "运镜"],
      canvas: null
    },
    {
      id: "w-cliff",
      title: "悬崖旅馆",
      author: "元片场",
      avatarHue: 265,
      likes: 27, liked: false, followed: false, views: 8900,
      category: "专业影视",
      cover: "assets/cover-cliff.jpg",
      video: "assets/video-flower.mp4",
      duration: "1:44", resolution: "2560 × 1440",
      publishedAt: "10月12日",
      desc: "云端之上，一间孤零零的旅馆。悬疑短片开场镜头。",
      promptTags: ["悬疑", "云海", "建筑", "黄昏"],
      canvas: null
    },
    {
      id: "w-tang",
      title: "朝代系列-唐《盛世长安》",
      author: "AIGC野生包",
      avatarHue: 30,
      likes: 39, liked: false, followed: false, views: 11200,
      category: "精选画布",
      cover: "assets/cover-tang.jpg",
      video: "assets/video-bbb.mp4",
      duration: "2:18", resolution: "3840 × 2160",
      publishedAt: "10月10日",
      desc: "朝代系列第一部：大唐长安的黄昏，灯火与宫阙。",
      promptTags: ["古风", "史诗", "长安", "航拍"],
      canvas: null
    },
    {
      id: "w-ancient",
      title: "古人开麦特别篇《过来人》——喆FILM",
      author: "喆FILM",
      avatarHue: 150,
      likes: 1048, liked: false, followed: false, views: 260000,
      category: "短剧漫剧",
      cover: "assets/cover-ancient.jpg",
      video: "assets/video-flower.mp4",
      duration: "4:40", resolution: "1920 × 1080",
      publishedAt: "10月8日",
      desc: "当古人拿到现代麦克风，会聊点什么？特别篇上线。",
      promptTags: ["喜剧", "综艺", "穿越", "脱口秀"],
      canvas: null
    }
  ]
};

/* 未配置专用画布的作品，按封面自动生成一套通用创作过程节点（演示用） */
DB.buildCanvas = function (work) {
  if (work.canvas) return work.canvas;
  const y = 100;
  return {
    nodes: [
      { id: "c1", type: "text", group: "创作记录", x: 0, y, w: 260, h: 140, title: "创意描述", text: work.desc },
      { id: "c2", type: "config", group: "创作记录", x: 0, y: y + 190, w: 260, h: 110, title: "生成参数", text: work.resolution + " · " + work.duration + "\n质量：电影级" },
      { id: "c3", type: "image", group: "创作记录", x: 300, y, w: 340, h: 255, title: "概念图", src: work.cover, prompt: work.promptTags.join(", ") },
      { id: "c4", type: "video", group: "创作记录", x: 680, y, w: 480, h: 270, title: "最终成片", src: work.video, poster: work.cover, resolution: work.resolution }
    ],
    edges: [
      { from: "c1", to: "c3" },
      { from: "c3", to: "c4" },
      { from: "c2", to: "c4" }
    ]
  };
};
