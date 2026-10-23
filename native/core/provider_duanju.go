package core

import (
	"net/url"
	"regexp"
	"strings"
)

const (
	sourceYaguo   = "yaguo"
	sourceMaoguo  = "maoguo"
	sourceFanguo  = "fanguo"
	sourceGuanguo = "guanguo"
	sourceHeguo   = "heguo"
	sourceXingguo = "xingguo"
	sourceHuaguo  = "huaguo"
	sourceNiuguo  = "niuguo"
	sourceWangguo = "wangguo"
	sourceFaguo   = "faguo"
	sourceWuguo   = "wuguo"
	sourceMiguo   = "miguo"
	sourceShuangguo = "shuangguo"
	sourceTaoguo = "taoguo"
	sourceYanguo = "yanguo"
	sourceYouguo = "youguo"
	sourceLiuguo = "liuguo"
	sourceMeiguo = "meiguo"
	sourceChengguo = "chengguo"
	sourceXiaoguo = "xiaoguo"
	sourceYingguo = "yingguo"
	sourceLuguo = "luguo"
	sourceLiguo = "liguo"
	sourceJuguo = "juguo"
	sourceZaoguo = "zaoguo"
	sourceNingguo = "ningguo"
	sourceMangguo = "mangguo"
	sourceChaoguo = "chaoguo"

	duanjuMaxBodyBytes = 8 * 1024 * 1024
	duanjuUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

var (
	yaguoBaseURL    = "https://app.whjzjx.cn"
	yaguoLoginURL   = "https://u.shytkjgs.com/user/v1/account/login"
	maoguoBaseURL   = "https://api-store.qmplaylet.com"
	maoguoReadURL   = "https://api-read.qmplaylet.com"
	fanguoBaseURL   = "https://xifan-api-cn.youlishipin.com"
	guanguoBaseURL  = "https://api.drama.9ddm.com"
	heguoBaseURL    = "https://www.kuaikaw.cn"
	xingguoBaseURL  = "http://read.api.duodutek.com"
	huaguoBaseURL   = "https://www.zywest263.com"
	niuguoBaseURL   = "https://ccc.chaojichaojichanga.com:35620"
	niuguoParseURL  = "http://ccs.jshh.gzbaoxian.com"
	niuguoParseURL2 = "http://101.42.92.211:5560"
	wangguoBaseURL  = "https://www.duanju2.com"
	faguoBaseURL    = "https://www.xzyx168.com"
	wuguoBaseURL    = "https://www.duanju55.com"
	miguoBaseURL    = "https://dmxq40.com"
	shuangguoBaseURL = "https://www.duanju2.com"
	taoguoBaseURL = "https://lujj31.buzz"
	yanguoBaseURL = "https://xqxq1.cc"
	youguoBaseURL = "https://youavhub.com"
	liuguoBaseURL = "https://www.llsp.me"
	meiguoBaseURL = "https://shiresm.lol"
	chengguoBaseURL = "https://a.qingyiduz.xyz"
	xiaoguoBaseURL = "https://yese.co"
	yingguoBaseURL = "https://missav02.xyz"
	luguoBaseURL = "https://luyitian.com"
	liguoBaseURL = "https://ririlu.cc"
	juguoBaseURL = "https://4tw3gy653a.bulunhufait.buzz"
	zaoguoBaseURL = "https://www.yhsp5.yachts/cn/home/web"
	ningguoBaseURL = "https://toptv15.cyou"
	mangguoBaseURL = "https://zsrqab03.zsrenqi.xyz"
	chaoguoBaseURL = "https://www.shanekids.com"
)

type duanjuSourceSpec struct {
	ID       string
	Name     string
	Base     string
	Kind     string
	Agent    string
	Referer  string
	Searcher bool
	Paged    bool
	Browser  bool
}

var duanjuProviderCatalog = []duanjuSourceSpec{
	{ID: sourceYaguo, Name: "芽果", Base: yaguoBaseURL, Kind: "api", Searcher: true, Paged: true},
	{ID: sourceShuangguo, Name: "爽果", Base: shuangguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceMaoguo, Name: "猫果", Base: maoguoBaseURL, Kind: "api", Searcher: true},
	{ID: sourceFanguo, Name: "饭果", Base: fanguoBaseURL, Kind: "api", Searcher: true},
	{ID: sourceGuanguo, Name: "观果", Base: guanguoBaseURL, Kind: "api", Agent: "okhttp/5.1.0", Searcher: true, Paged: true},
	{ID: sourceHeguo, Name: "河果", Base: heguoBaseURL, Kind: "next", Searcher: true},
	{ID: sourceXingguo, Name: "星果", Base: xingguoBaseURL, Kind: "api", Searcher: true},
	{ID: sourceHuaguo, Name: "花果", Base: huaguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceNiuguo, Name: "牛果", Base: niuguoBaseURL, Kind: "api", Searcher: true, Paged: true},
	{ID: sourceWangguo, Name: "网果", Base: wangguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceFaguo, Name: "发果", Base: faguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceWuguo, Name: "伍果", Base: wuguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceMiguo, Name: "米果", Base: miguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceTaoguo, Name: "桃果 🔞", Base: taoguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceYanguo, Name: "艷果 🔞", Base: yanguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceYouguo, Name: "柚果 🔞", Base: youguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceLiuguo, Name: "榴果 🔞", Base: liuguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceMeiguo, Name: "美果 🔞", Base: meiguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceChengguo, Name: "城果 🔞", Base: chengguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceXiaoguo, Name: "宵果 🔞", Base: xiaoguoBaseURL, Kind: "maccms", Searcher: true, Paged: true},
	{ID: sourceYingguo, Name: "櫻果 🔞", Base: yingguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceLuguo, Name: "露果 🔞", Base: luguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceLiguo, Name: "荔果 🔞", Base: liguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceJuguo, Name: "橘果 🔞", Base: juguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceZaoguo, Name: "棗果 🔞", Base: zaoguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceNingguo, Name: "檸果 🔞", Base: ningguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceMangguo, Name: "芒果 🔞", Base: mangguoBaseURL, Kind: "maccms", Searcher: true, Paged: false},
	{ID: sourceChaoguo, Name: "超果 🔞", Base: chaoguoBaseURL, Kind: "web", Searcher: true, Paged: true},
}

var duanjuMarkupPattern = regexp.MustCompile(`<[^>]{1,80}>`)

func duanjuPlainText(raw string) string {
	text := strings.TrimSpace(raw)
	if !strings.Contains(text, "<") {
		return text
	}
	text = duanjuMarkupPattern.ReplaceAllString(text, "")
	text = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(text)
	return strings.Join(strings.Fields(text), " ")
}

const (
	heguoSearchPath       = "/seo/video/6007"
	heguoSearchPName      = "www.kuaikaw.cn"
	heguoSearchSourceType = 1
)

var duanjuSourceAliases = map[string]string{
	"yaguo": "yaguo", "xingya": sourceYaguo, "星芽": sourceYaguo, "app.whjzjx.cn": sourceYaguo,
	"maoguo": "maoguo", "qimao": sourceMaoguo, "七猫": sourceMaoguo, "api-store.qmplaylet.com": sourceMaoguo,
	"fanguo": "fanguo", "xifan": sourceFanguo, "西饭": sourceFanguo, "xifan-api-cn.youlishipin.com": sourceFanguo,
	"guanguo": "guanguo", "weiguan": sourceGuanguo, "围观": sourceGuanguo, "api.drama.9ddm.com": sourceGuanguo,
	"heguo": "heguo", "hema": sourceHeguo, "河马": sourceHeguo, "www.kuaikaw.cn": sourceHeguo, "kuaikaw.cn": sourceHeguo,
	"xingguo": "xingguo", "xingxing": sourceXingguo, "星星": sourceXingguo, "read.api.duodutek.com": sourceXingguo,
	"huaguo": "huaguo", "huasheng": sourceHuaguo, "花生": sourceHuaguo, "www.zywest263.com": sourceHuaguo,
	"niuguo": "niuguo", "niuniu": sourceNiuguo, "牛牛": sourceNiuguo,
	"wangguo": "wangguo", "duanjuwangzhan": sourceWangguo, "短剧网站": sourceWangguo, "www.duanju2.com": sourceWangguo,
	"faguo": "faguo", "168": sourceFaguo, "xzyx168": sourceFaguo, "www.xzyx168.com": sourceFaguo,
	"wuguo": "wuguo", "wuwu": sourceWuguo, "五五": sourceWuguo, "www.duanju55.com": sourceWuguo,
	"shuangguo": sourceShuangguo, "爽果": sourceShuangguo,
	"miguo": sourceMiguo, "米果": sourceMiguo, "dmxq40.com": sourceMiguo, "www.dmxq40.com": sourceMiguo,
	"taoguo": sourceTaoguo, "桃果": sourceTaoguo, "lujj31.buzz": sourceTaoguo,
	"yanguo": sourceYanguo, "艷果": sourceYanguo, "艳果": sourceYanguo, "xqxq1.cc": sourceYanguo,
	"youguo": sourceYouguo, "柚果": sourceYouguo, "youavhub.com": sourceYouguo, "www.youavhub.com": sourceYouguo,
	"liuguo": sourceLiuguo, "榴果": sourceLiuguo, "llsp.me": sourceLiuguo, "www.llsp.me": sourceLiuguo,
"meiguo": sourceMeiguo, "美果": sourceMeiguo, "shiresm.lol": sourceMeiguo,
"chengguo": sourceChengguo, "城果": sourceChengguo, "a.qingyiduz.xyz": sourceChengguo,
"xiaoguo": sourceXiaoguo, "宵果": sourceXiaoguo, "yese.co": sourceXiaoguo,
"yingguo": sourceYingguo, "櫻果": sourceYingguo, "missav02.xyz": sourceYingguo,
"luguo": sourceLuguo, "露果": sourceLuguo, "luyitian.com": sourceLuguo,
"liguo": sourceLiguo, "荔果": sourceLiguo, "ririlu.cc": sourceLiguo,
"juguo": sourceJuguo, "橘果": sourceJuguo, "4tw3gy653a.bulunhufait.buzz": sourceJuguo,
"zaoguo": sourceZaoguo, "棗果": sourceZaoguo, "www.yhsp5.yachts": sourceZaoguo,
"ningguo": sourceNingguo, "檸果": sourceNingguo, "toptv15.cyou": sourceNingguo,
"mangguo": sourceMangguo, "芒果": sourceMangguo, "zsrqab03.zsrenqi.xyz": sourceMangguo,
"chaoguo": sourceChaoguo, "超果": sourceChaoguo, "www.shanekids.com": sourceChaoguo,
}

var duanjuSourcesByName = func() map[string]duanjuSourceSpec {
	registry := map[string]duanjuSourceSpec{}
	for _, spec := range duanjuProviderCatalog {
		registry[spec.ID] = spec
	}
	return registry
}()

func isDuanjuProviderSource(source string) bool {
	_, found := duanjuSourcesByName[canonicalProviderSource(source)]
	if found {
		return true
	}
	_, found = customMaccmsSpecFor(canonicalProviderSource(source))
	return found
}

func duanjuSourceSpecFor(source string) (duanjuSourceSpec, bool) {
	source = canonicalProviderSource(source)
	spec, found := duanjuSourcesByName[source]
	if found {
		return spec, found
	}
	return customMaccmsSpecFor(source)
}

func duanjuSourceName(source string) string {
	if spec, found := duanjuSourceSpecFor(source); found {
		return spec.Name
	}
	return canonicalProviderSource(source)
}

func (d *Downloader) duanjuBaseURL(source string) string {
	spec, found := duanjuSourceSpecFor(source)
	if !found {
		return ""
	}
	if configured := d.providerHosts[canonicalProviderSource(source)]; configured != "" {
		return strings.TrimRight(configured, "/")
	}
	return strings.TrimRight(spec.Base, "/")
}

func duanjuSourceForHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	switch {
	case host == "app.whjzjx.cn" || host == "u.shytkjgs.com":
		return sourceYaguo
	case host == "api-store.qmplaylet.com" || host == "api-read.qmplaylet.com":
		return sourceMaoguo
	case host == "xifan-api-cn.youlishipin.com":
		return sourceFanguo
	case host == "api.drama.9ddm.com":
		return sourceGuanguo
	case host == "www.kuaikaw.cn" || host == "kuaikaw.cn":
		return sourceHeguo
	case host == "read.api.duodutek.com":
		return sourceXingguo
	case host == "www.zywest263.com" || host == "zywest263.com":
		return sourceHuaguo
	case host == "ccc.chaojichaojichanga.com":
		return sourceNiuguo
	case host == "www.duanju2.com" || host == "duanju2.com":
		return sourceWangguo
	case host == "www.xzyx168.com" || host == "xzyx168.com":
		return sourceFaguo
	case host == "www.duanju55.com" || host == "duanju55.com":
		return sourceWuguo
	case host == "dmxq40.com" || host == "www.dmxq40.com":
		return sourceMiguo
	case host == "lujj31.buzz" || host == "www.lujj31.buzz":
		return sourceTaoguo
	case host == "xqxq1.cc" || host == "www.xqxq1.cc":
		return sourceYanguo
	case host == "youavhub.com" || host == "www.youavhub.com":
		return sourceYouguo
	case host == "llsp.me" || host == "www.llsp.me":
		return sourceLiuguo
	case host == "shiresm.lol":
		return sourceMeiguo
	case host == "a.qingyiduz.xyz":
		return sourceChengguo
	case host == "yese.co":
		return sourceXiaoguo
	case host == "missav02.xyz":
		return sourceYingguo
	case host == "luyitian.com":
		return sourceLuguo
	case host == "ririlu.cc":
		return sourceLiguo
	case host == "4tw3gy653a.bulunhufait.buzz":
		return sourceJuguo
	case host == "www.yhsp5.yachts":
		return sourceZaoguo
	case host == "toptv15.cyou":
		return sourceNingguo
	case host == "zsrqab03.zsrenqi.xyz":
		return sourceMangguo
	case host == "www.shanekids.com":
		return sourceChaoguo
	default:
		return ""
	}
}

func validDuanjuCategory(source, category string) bool {
	if category == "" {
		return true
	}
	if len(category) > 128 || strings.ContainsAny(category, "|\\\x00\r\n") {
		return false
	}
	if isCustomMaccmsSource(source) {
		// 自定义 MacCMS 源分类 ID 允许路径形式（含 /），仅做基础非法字符校验。
		return true
	}
	switch canonicalProviderSource(source) {
	case sourceHeguo, sourceXingguo, sourceYaguo, sourceMaoguo, sourceNiuguo:
		return webProviderNumericID.MatchString(strings.SplitN(category, "-", 2)[0])
	case sourceFanguo, sourceGuanguo:
		return true
	default:
		return true
	}
}

type duanjuCategory struct {
	ID   string
	Name string
}

var duanjuStaticCategories = map[string][]duanjuCategory{
	sourceShuangguo: {
		{ID: "all", Name: "全部"},
		{ID: "女频恋爱", Name: "女頻戀愛"},
		{ID: "脑洞悬疑", Name: "腦洞懸疑"},
		{ID: "年代穿越", Name: "年代穿越"},
		{ID: "古装仙侠", Name: "古裝仙俠"},
		{ID: "现代都市", Name: "現代都市"},
		{ID: "反转", Name: "反轉"},
		{ID: "爽文", Name: "爽文"},
		{ID: "短剧", Name: "短劇"},
	},
	sourceMeiguo: {{ID: "4", Name: "全部"}},
	sourceChengguo: {{ID: "1", Name: "全部"}},
	sourceXiaoguo: {{ID: "1", Name: "全部"}},
	sourceYingguo: {{ID: "28", Name: "全部"}},
	sourceLuguo: {{ID: "26", Name: "全部"}},
	sourceLiguo: {{ID: "28", Name: "全部"}},
	sourceJuguo: {{ID: "66", Name: "全部"}},
	sourceZaoguo: {{ID: "21", Name: "全部"}},
	sourceNingguo: {{ID: "1", Name: "全部"}},
	sourceMangguo: {{ID: "20", Name: "全部"}},

	sourceTaoguo: {{ID: "1", Name: "全部"}},
	sourceYanguo: {{ID: "1", Name: "全部"}},
	sourceYaguo: {
		{ID: "1", Name: "剧场"},
		{ID: "9", Name: "榜单"},
	},
	sourceMaoguo: {
		{ID: "0", Name: "推荐"},
	},
	sourceFanguo: {
		{ID: "都市", Name: "都市"},
		{ID: "甜宠", Name: "甜宠"},
		{ID: "逆袭", Name: "逆袭"},
		{ID: "战神", Name: "战神"},
		{ID: "古装", Name: "古装"},
		{ID: "穿越", Name: "穿越"},
		{ID: "萌宝", Name: "萌宝"},
	},
	sourceXingguo: {
		{ID: "1287", Name: "推荐一"},
		{ID: "1288", Name: "推荐二"},
		{ID: "1289", Name: "推荐三"},
		{ID: "1290", Name: "推荐四"},
		{ID: "1291", Name: "推荐五"},
	},
	sourceNiuguo: {
		{ID: "1", Name: "短剧"},
		{ID: "2", Name: "电影"},
		{ID: "3", Name: "电视剧"},
		{ID: "4", Name: "动漫"},
		{ID: "5", Name: "综艺"},
	},
	sourceHeguo: {
		{ID: "462", Name: "甜宠"},
		{ID: "1102", Name: "古装仙侠"},
		{ID: "1145", Name: "现代言情"},
		{ID: "1170", Name: "青春"},
		{ID: "585", Name: "豪门恩怨"},
		{ID: "417-464", Name: "逆袭"},
		{ID: "439-465", Name: "重生"},
		{ID: "1159", Name: "系统"},
		{ID: "1147", Name: "总裁"},
		{ID: "943", Name: "职场商战"},
	},
	sourceHuaguo: {
		{ID: "all", Name: "全部"},
		{ID: "27", Name: "短剧"},
	},
}

func duanjuQueryEscape(value string) string { return url.QueryEscape(strings.TrimSpace(value)) }

func duanjuAbsolute(base, reference string) string {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return ""
	}
	if strings.HasPrefix(reference, "//") {
		return "https:" + reference
	}
	address, err := url.Parse(reference)
	if err != nil {
		return ""
	}
	if address.IsAbs() {
		return address.String()
	}
	origin, err := url.Parse(strings.TrimRight(base, "/") + "/")
	if err != nil {
		return ""
	}
	return origin.ResolveReference(address).String()
}
