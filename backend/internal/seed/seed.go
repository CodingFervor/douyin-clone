package seed

import (
	"database/sql"
	"log"

	"github.com/CodingFervor/douyin-clone/backend/internal/model"
	"github.com/CodingFervor/douyin-clone/backend/internal/repository"
)

// Run seeds users, videos, and comments if empty (idempotent).
func Run(db *sql.DB) {
	userRepo := repository.NewUserRepo(db)
	videoRepo := repository.NewVideoRepo(db)
	commentRepo := repository.NewCommentRepo(db)

	if u, _ := userRepo.FindByUsername("admin"); u == nil && !userRepo.Exists("admin") {
		users := []model.User{
			{Username: "admin", Password: "admin123", Nickname: "抖音小助手", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=admin", Bio: "官方小助手 🎵"},
			{Username: "traveler", Password: "123456", Nickname: "旅行的意义", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=travel", Bio: "用镜头记录世界 ✈️ 环球旅行博主"},
			{Username: "foodie", Password: "123456", Nickname: "吃货日记", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=food", Bio: "寻找城市里的美食 🍜 美食达人"},
			{Username: "dancer", Password: "123456", Nickname: "舞动青春", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=dance", Bio: "舞蹈博主 | 每日更新"},
			{Username: "petlover", Password: "123456", Nickname: "萌宠乐园", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=pet", Bio: "猫狗双全 🐱🐶 萌宠日常"},
		}
		for i := range users {
			if err := userRepo.Create(&users[i]); err != nil {
				log.Printf("seed user: %v", err)
			}
		}
		// set some follower/like counts
		_, _ = db.Exec(`UPDATE users SET followers_count=?, likes_count=? WHERE id=2`, 1280000, 8900000)
		_, _ = db.Exec(`UPDATE users SET followers_count=?, likes_count=? WHERE id=3`, 890000, 5200000)
		_, _ = db.Exec(`UPDATE users SET followers_count=?, likes_count=? WHERE id=4`, 2300000, 15000000)
		_, _ = db.Exec(`UPDATE users SET followers_count=?, likes_count=? WHERE id=5`, 560000, 3200000)
	}

	var vidCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM videos`).Scan(&vidCount)
	if vidCount == 0 {
		// Use Google's sample videos (publicly available test clips) as mock content.
		const sample1 = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4"
		const sample2 = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerEscapes.mp4"
		const sample3 = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerFun.mp4"
		const sample4 = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerJoyrides.mp4"
		const sample5 = "https://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerMeltdowns.mp4"
		const cover = "https://picsum.photos/seed/dy{ID}/750/1334"

		videos := []model.VideoInput{
			{Title: "冰岛极光之旅，美到窒息！", Description: "在冰岛追极光的第三天，终于拍到了！大自然太神奇了 🌌", VideoURL: sample1, CoverURL: cover, Duration: 15, Tags: "旅行,冰岛,极光", Music: "原声 - 旅行的意义"},
			{Title: "环游世界30天，最治愈的瞬间", Description: "从冰岛到新西兰，每一帧都是壁纸 ✨", VideoURL: sample2, CoverURL: cover, Duration: 12, Tags: "旅行,vlog,治愈", Music: "Sunset Vibes"},
			{Title: "这家隐秘小店的面太好吃了", Description: "排队2小时也值得！正宗日式拉面 🍜", VideoURL: sample3, CoverURL: cover, Duration: 18, Tags: "美食,探店,拉面", Music: "吃货BGM"},
			{Title: "5分钟学会这支舞", Description: "超简单的流行舞教学，跟着跳起来！💃", VideoURL: sample4, CoverURL: cover, Duration: 20, Tags: "舞蹈,教学,流行", Music: "动感舞曲"},
			{Title: "我家猫第一次看到雪", Description: "萌化了！主子一脸懵的样子太可爱 🐱❄️", VideoURL: sample5, CoverURL: cover, Duration: 10, Tags: "萌宠,猫咪,搞笑", Music: "可爱原声"},
			{Title: "海边日落延时摄影", Description: "等了三个小时拍到的绝美日落 🌅", VideoURL: sample1, CoverURL: cover, Duration: 14, Tags: "摄影,风景,日落", Music: "Calm Ocean"},
			{Title: "深夜放毒·家常红烧肉", Description: "肥而不腻入口即化，秘方全公开！", VideoURL: sample2, CoverURL: cover, Duration: 22, Tags: "美食,教程,家常菜", Music: "厨房BGM"},
			{Title: "街舞battle现场燃炸了", Description: "地下街舞比赛高光时刻 🔥", VideoURL: sample3, CoverURL: cover, Duration: 25, Tags: "舞蹈,街舞,battle", Music: "Hip Hop Beat"},
			{Title: "狗狗第一次去海边", Description: "金毛在沙滩上撒欢的样子太治愈了 🐶", VideoURL: sample4, CoverURL: cover, Duration: 11, Tags: "萌宠,狗狗,海边", Music: "Happy Day"},
			{Title: "一个人旅行安全吗？我的经验", Description: "女生独行20国的安全建议 ✈️", VideoURL: sample5, CoverURL: cover, Duration: 30, Tags: "旅行,攻略,独行", Music: "旅行原声"},
		}
		for i := range videos {
			authorID := int64(i%4) + 2 // authors: traveler, foodie, dancer, petlover
			id, err := videoRepo.Create(&videos[i], authorID)
			if err != nil {
				log.Printf("seed video %d: %v", i, err)
				continue
			}
			// backfill the cover placeholder with the real id
			_, _ = db.Exec(`UPDATE videos SET cover_url=? WHERE id=?`, coverVideo(id), id)
			// randomize engagement
			plays := 50000 + i*37891
			likes := 2000 + i*1453
			comments := 100 + i*37
			shares := 300 + i*89
			_, _ = db.Exec(`UPDATE videos SET plays=?, likes=?, comments_count=?, shares=? WHERE id=?`, plays, likes, comments, shares, id)
		}
	}

	// Seed a few comments
	if n, _ := commentRepo.Count(); n == 0 {
		comments := []model.Comment{
			{VideoID: 1, UserID: 1, Username: "抖音小助手", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=admin", Content: "这也太美了吧！求定位 📍"},
			{VideoID: 1, UserID: 3, Username: "吃货日记", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=food", Content: "已加入心愿清单，太治愈了"},
			{VideoID: 3, UserID: 2, Username: "旅行的意义", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=travel", Content: "看饿了，这家店在哪？"},
			{VideoID: 5, UserID: 1, Username: "抖音小助手", Avatar: "https://api.dicebear.com/7.x/fun-emoji/svg?seed=admin", Content: "哈哈哈哈主子表情绝了"},
		}
		for _, c := range comments {
			_ = commentRepo.Create(&c)
		}
	}

	log.Println("seed: douyin mock data ensured")
}

func coverVideo(id int64) string {
	return "https://picsum.photos/seed/dy" + itoa(int(id)) + "/750/1334"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [12]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
