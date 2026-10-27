/* ══════════ API 存根层（形状即未来真实接口，接后端时替换实现即可） ══════════
 * 响应结构与项目后端规范一致：{ code, data, msg }
 * GET  /api/v1/showcase?category=&keyword=      → 作品列表
 * GET  /api/v1/showcase/:id                     → 作品详情
 * GET  /api/v1/showcase/:id/process             → 创作过程画布数据
 * POST /api/v1/showcase/:id/like                → 点赞/取消
 * POST /api/v1/showcase/:id/fork                → 复制到我的画布
 */

function delay(ms) {
  return new Promise(function (resolve) { setTimeout(resolve, ms); });
}

function ok(data) {
  return { code: 0, data: data, msg: "ok" };
}

async function fetchShowcase(_ref) {
  var category = (_ref || {}).category, keyword = (_ref || {}).keyword;
  await delay(350); // 模拟网络延迟，让 loading 状态可见
  var list = DB.works.filter(function (w) {
    var byCat = !category || category === "全部" || w.category === category;
    var kw = (keyword || "").trim().toLowerCase();
    var byKw = !kw || w.title.toLowerCase().includes(kw) || w.author.toLowerCase().includes(kw);
    return byCat && byKw;
  });
  return ok(list);
}

async function fetchWorkDetail(id) {
  await delay(200);
  var work = DB.works.find(function (w) { return w.id === id; });
  return work ? ok(work) : { code: 404, data: null, msg: "作品不存在" };
}

async function fetchWorkProcess(id) {
  await delay(450); // 画布数据稍大，延迟更明显
  var work = DB.works.find(function (w) { return w.id === id; });
  return work ? ok(DB.buildCanvas(work)) : { code: 404, data: null, msg: "作品不存在" };
}

async function toggleLike(id) {
  await delay(150);
  var work = DB.works.find(function (w) { return w.id === id; });
  if (!work) return { code: 404, data: null, msg: "作品不存在" };
  work.liked = !work.liked;
  work.likes += work.liked ? 1 : -1;
  return ok({ liked: work.liked, likes: work.likes });
}

async function toggleFollow(id) {
  await delay(150);
  var work = DB.works.find(function (w) { return w.id === id; });
  if (!work) return { code: 404, data: null, msg: "作品不存在" };
  work.followed = !work.followed;
  return ok({ followed: work.followed });
}

async function forkWork(id) {
  await delay(600); // 复制项目较慢，符合真实体验
  var work = DB.works.find(function (w) { return w.id === id; });
  if (!work) return { code: 404, data: null, msg: "作品不存在" };
  return ok({ projectId: "fork-" + id, title: work.title + "（副本）" });
}
