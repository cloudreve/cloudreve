package inventory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudreve/Cloudreve/v4/ent"
	"github.com/cloudreve/Cloudreve/v4/ent/setting"
	"github.com/cloudreve/Cloudreve/v4/inventory/types"
	"github.com/cloudreve/Cloudreve/v4/pkg/cache"
	"github.com/cloudreve/Cloudreve/v4/pkg/util"
	"github.com/gofrs/uuid"
)

type (
	SettingClient interface {
		TxOperator
		// Get gets a setting value from DB, returns error if setting cannot be found.
		Get(ctx context.Context, name string) (string, error)
		// Set sets a setting value to DB.
		Set(ctx context.Context, settings map[string]string) error
		// Gets gets multiple setting values from DB, returns error if any setting cannot be found.
		Gets(ctx context.Context, names []string) (map[string]string, error)
	}
)

// NewSettingClient creates a new SettingClient
func NewSettingClient(client *ent.Client, kv cache.Driver) SettingClient {
	return &settingClient{client: client, kv: kv}
}

type settingClient struct {
	client *ent.Client
	kv     cache.Driver
}

// SetClient sets the client for the setting client
func (c *settingClient) SetClient(newClient *ent.Client) TxOperator {
	return &settingClient{client: newClient, kv: c.kv}
}

// GetClient gets the client for the setting client
func (c *settingClient) GetClient() *ent.Client {
	return c.client
}

func (c *settingClient) Get(ctx context.Context, name string) (string, error) {
	s, err := c.client.Setting.Query().Where(setting.Name(name)).Only(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query setting %q from DB: %w", name, err)
	}

	return s.Value, nil
}

func (c *settingClient) Gets(ctx context.Context, names []string) (map[string]string, error) {
	settings := make(map[string]string)
	res, err := c.client.Setting.Query().Where(setting.NameIn(names...)).All(ctx)
	if err != nil {
		return nil, err
	}

	for _, s := range res {
		settings[s.Name] = s.Value
	}

	return settings, nil
}

func (c *settingClient) Set(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		if err := c.client.Setting.Update().Where(setting.Name(k)).SetValue(v).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create setting %q: %w", k, err)
		}

	}

	return nil
}

var (
	defaultIcons = []types.FileTypeIconSetting{
		{
			Exts:  []string{"mp3", "flac", "ape", "wav", "acc", "ogg", "m4a"},
			Icon:  "audio",
			Color: "#651fff",
		},
		{
			Exts:  []string{"m3u8", "mp4", "flv", "avi", "wmv", "mkv", "rm", "rmvb", "mov", "ogv"},
			Icon:  "video",
			Color: "#d50000",
		},
		{
			Exts:  []string{"bmp", "iff", "png", "gif", "jpg", "jpeg", "psd", "svg", "webp", "heif", "heic", "tiff", "avif"},
			Icon:  "image",
			Color: "#d32f2f",
		},
		{
			Exts:  []string{"3fr", "ari", "arw", "bay", "braw", "crw", "cr2", "cr3", "cap", "dcs", "dcr", "dng", "drf", "eip", "erf", "fff", "gpr", "iiq", "k25", "kdc", "mdc", "mef", "mos", "mrw", "nef", "nrw", "obm", "orf", "pef", "ptx", "pxn", "r3d", "raf", "raw", "rwl", "rw2", "rwz", "sr2", "srf", "srw", "tif", "x3f"},
			Icon:  "raw",
			Color: "#d32f2f",
		},
		{
			Exts:  []string{"pdf"},
			Color: "#f44336",
			Icon:  "pdf",
		},
		{
			Exts:  []string{"doc", "docx"},
			Color: "#538ce5",
			Icon:  "word",
		},
		{
			Exts:  []string{"ppt", "pptx"},
			Color: "#EF633F",
			Icon:  "ppt",
		},
		{
			Exts:  []string{"xls", "xlsx", "csv"},
			Color: "#4caf50",
			Icon:  "excel",
		},
		{
			Exts:  []string{"txt", "html"},
			Color: "#607d8b",
			Icon:  "text",
		},
		{
			Exts:  []string{"torrent"},
			Color: "#5c6bc0",
			Icon:  "torrent",
		},
		{
			Exts:  []string{"zip", "gz", "xz", "tar", "rar", "7z", "bz2", "z"},
			Color: "#f9a825",
			Icon:  "zip",
		},
		{
			Exts:  []string{"exe", "msi"},
			Color: "#1a237e",
			Icon:  "exe",
		},
		{
			Exts:  []string{"apk"},
			Color: "#8bc34a",
			Icon:  "android",
		},
		{
			Exts:  []string{"go"},
			Color: "#16b3da",
			Icon:  "go",
		},
		{
			Exts:  []string{"py"},
			Color: "#3776ab",
			Icon:  "python",
		},
		{
			Exts:  []string{"c"},
			Color: "#a4c639",
			Icon:  "c",
		},
		{
			Exts:  []string{"cpp"},
			Color: "#f34b7d",
			Icon:  "cpp",
		},
		{
			Exts:  []string{"js", "jsx"},
			Color: "#f4d003",
			Icon:  "js",
		},
		{
			Exts:  []string{"epub"},
			Color: "#81b315",
			Icon:  "book",
		},
		{
			Exts:      []string{"rs"},
			Color:     "#000",
			ColorDark: "#fff",
			Icon:      "rust",
		},
		{
			Exts:  []string{"drawio"},
			Color: "#F08705",
			Icon:  "flowchart",
		},
		{
			Exts:  []string{"dwb"},
			Color: "#F08705",
			Icon:  "whiteboard",
		},
		{
			Exts:      []string{"md"},
			Color:     "#383838",
			ColorDark: "#cbcbcb",
			Icon:      "markdown",
		},
		{
			Img:  "/static/img/viewers/excalidraw.svg",
			Exts: []string{"excalidraw"},
		},
	}

	defaultFileViewers = []types.ViewerGroup{
		{
			Viewers: []types.Viewer{
				{
					ID:          "music",
					Type:        types.ViewerTypeBuiltin,
					DisplayName: "fileManager.musicPlayer",
					Exts:        []string{"mp3", "ogg", "wav", "flac", "m4a"},
				},
				{
					ID:          "epub",
					Type:        types.ViewerTypeBuiltin,
					DisplayName: "fileManager.epubViewer",
					Exts:        []string{"epub"},
				},
				{
					ID:          "googledocs",
					Type:        types.ViewerTypeCustom,
					DisplayName: "fileManager.googledocs",
					Icon:        "/static/img/viewers/gdrive.png",
					Url:         "https://docs.google.com/gview?url={$src}&embedded=true",
					Exts:        []string{"jpeg", "png", "gif", "tiff", "bmp", "webm", "mpeg4", "3gpp", "mov", "avi", "mpegps", "wmv", "flv", "txt", "css", "html", "php", "c", "cpp", "h", "hpp", "js", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "pdf", "pages", "ai", "psd", "tiff", "dxf", "svg", "eps", "ps", "ttf", "xps"},
					MaxSize:     26214400,
				},
				{
					ID:          "m365online",
					Type:        types.ViewerTypeCustom,
					DisplayName: "fileManager.m365viewer",
					Icon:        "/static/img/viewers/m365.svg",
					Url:         "https://view.officeapps.live.com/op/view.aspx?src={$src}",
					Exts:        []string{"doc", "docx", "docm", "dotm", "dotx", "xlsx", "xlsb", "xls", "xlsm", "pptx", "ppsx", "ppt", "pps", "pptm", "potm", "ppam", "potx", "ppsm"},
					MaxSize:     10485760,
				},
				{
					ID:          "pdf",
					Type:        types.ViewerTypeBuiltin,
					DisplayName: "fileManager.pdfViewer",
					Exts:        []string{"pdf"},
				},
				{
					ID:          "video",
					Type:        types.ViewerTypeBuiltin,
					Icon:        "/static/img/viewers/artplayer.png",
					DisplayName: "Artplayer",
					Exts:        []string{"mp4", "mkv", "webm", "avi", "mov", "m3u8", "flv"},
				},
				{
					ID:          "markdown",
					Type:        types.ViewerTypeBuiltin,
					DisplayName: "fileManager.markdownEditor",
					Exts:        []string{"md"},
					Templates: []types.NewFileTemplate{
						{
							Ext:         "md",
							DisplayName: "Markdown",
						},
					},
				},
				{
					ID:          "drawio",
					Type:        types.ViewerTypeBuiltin,
					Icon:        "/static/img/viewers/drawio.svg",
					DisplayName: "draw.io",
					Exts:        []string{"drawio", "dwb"},
					Props: map[string]string{
						"host": "https://embed.diagrams.net",
					},
					Templates: []types.NewFileTemplate{
						{
							Ext:         "drawio",
							DisplayName: "fileManager.diagram",
						},
						{
							Ext:         "dwb",
							DisplayName: "fileManager.whiteboard",
						},
					},
				},
				{
					ID:          "image",
					Type:        types.ViewerTypeBuiltin,
					DisplayName: "fileManager.imageViewer",
					Exts:        []string{"bmp", "png", "gif", "jpg", "jpeg", "svg", "webp", "heic", "heif"},
				},
				{
					ID:          "monaco",
					Type:        types.ViewerTypeBuiltin,
					Icon:        "/static/img/viewers/monaco.svg",
					DisplayName: "fileManager.monacoEditor",
					Exts:        []string{"md", "txt", "json", "php", "py", "bat", "c", "h", "cpp", "hpp", "cs", "css", "dockerfile", "go", "html", "htm", "ini", "java", "js", "jsx", "less", "lua", "sh", "sql", "xml", "yaml"},
					Templates: []types.NewFileTemplate{
						{
							Ext:         "txt",
							DisplayName: "fileManager.text",
						},
					},
				},
				{
					ID:          "photopea",
					Type:        types.ViewerTypeBuiltin,
					Icon:        "/static/img/viewers/photopea.png",
					DisplayName: "Photopea",
					Exts:        []string{"psd", "ai", "indd", "xcf", "xd", "fig", "kri", "clip", "pxd", "pxz", "cdr", "ufo", "afphoyo", "svg", "esp", "pdf", "pdn", "wmf", "emf", "png", "jpg", "jpeg", "gif", "webp", "ico", "icns", "bmp", "avif", "heic", "jxl", "ppm", "pgm", "pbm", "tiff", "dds", "iff", "anim", "tga", "dng", "nef", "cr2", "cr3", "arw", "rw2", "raf", "orf", "gpr", "3fr", "fff"},
				},
				{
					ID:          "excalidraw",
					Type:        types.ViewerTypeBuiltin,
					Icon:        "/static/img/viewers/excalidraw.svg",
					DisplayName: "Excalidraw",
					Exts:        []string{"excalidraw"},
					Templates: []types.NewFileTemplate{
						{
							Ext:         "excalidraw",
							DisplayName: "Excalidraw",
						},
					},
				},
			},
		},
	}
)

var DefaultSettings = map[string]string{
	"siteURL":                                    `http://localhost:5212`,
	"siteName":                                   `Cloudreve`,
	"siteDes":                                    "Cloudreve",
	"siteID":                                     uuid.Must(uuid.NewV4()).String(),
	"siteTitle":                                  "Cloud storage for everyone",
	"siteScript":                                 "",
	"pwa_small_icon":                             "/static/img/favicon.ico",
	"pwa_medium_icon":                            "/static/img/logo192.png",
	"pwa_large_icon":                             "/static/img/logo512.png",
	"pwa_display":                                "standalone",
	"pwa_theme_color":                            "#000000",
	"pwa_background_color":                       "#ffffff",
	"register_enabled":                           `1`,
	"default_group":                              `2`,
	"fromName":                                   `Cloudreve`,
	"mail_keepalive":                             `30`,
	"fromAdress":                                 `no-reply@cloudreve.org`,
	"smtpHost":                                   `smtp.cloudreve.com`,
	"smtpPort":                                   `25`,
	"replyTo":                                    `support@cloudreve.org`,
	"smtpUser":                                   `smtp.cloudreve.com`,
	"smtpPass":                                   ``,
	"smtpEncryption":                             `0`,
	"ban_time":                                   `604800`,
	"maxEditSize":                                `52428800`,
	"archive_timeout":                            `600`,
	"upload_session_timeout":                     `86400`,
	"slave_api_timeout":                          `60`,
	"folder_props_timeout":                       `300`,
	"chunk_retries":                              `5`,
	"use_temp_chunk_buffer":                      `1`,
	"login_captcha":                              `0`,
	"reg_captcha":                                `0`,
	"email_active":                               `0`,
	"forget_captcha":                             `0`,
	"gravatar_server":                            `https://www.gravatar.com/`,
	"defaultTheme":                               `#1976d2`,
	"theme_options":                              `{"#1976d2":{"light":{"palette":{"primary":{"main":"#1976d2","light":"#42a5f5","dark":"#1565c0"},"secondary":{"main":"#9c27b0","light":"#ba68c8","dark":"#7b1fa2"}}},"dark":{"palette":{"primary":{"main":"#90caf9","light":"#e3f2fd","dark":"#42a5f5"},"secondary":{"main":"#ce93d8","light":"#f3e5f5","dark":"#ab47bc"}}}},"#3f51b5":{"light":{"palette":{"primary":{"main":"#3f51b5"},"secondary":{"main":"#f50057"}}},"dark":{"palette":{"primary":{"main":"#9fa8da"},"secondary":{"main":"#ff4081"}}}}}`,
	"max_parallel_transfer":                      `4`,
	"secret_key":                                 util.RandStringRunes(256),
	"temp_path":                                  "temp",
	"avatar_path":                                "avatar",
	"avatar_size":                                "4194304",
	"avatar_size_l":                              "200",
	"cron_garbage_collect":                       "@every 30m",
	"cron_entity_collect":                        "@every 15m",
	"cron_trash_bin_collect":                     "@every 33m",
	"cron_oauth_cred_refresh":                    "@every 230h",
	"authn_enabled":                              "1",
	"captcha_type":                               "normal",
	"captcha_height":                             "60",
	"captcha_width":                              "240",
	"captcha_mode":                               "3",
	"captcha_ComplexOfNoiseText":                 "0",
	"captcha_ComplexOfNoiseDot":                  "0",
	"captcha_IsShowHollowLine":                   "0",
	"captcha_IsShowNoiseDot":                     "1",
	"captcha_IsShowNoiseText":                    "0",
	"captcha_IsShowSlimeLine":                    "1",
	"captcha_IsShowSineLine":                     "0",
	"captcha_CaptchaLen":                         "6",
	"captcha_ReCaptchaKey":                       "defaultKey",
	"captcha_ReCaptchaSecret":                    "defaultSecret",
	"captcha_turnstile_site_key":                 "",
	"captcha_turnstile_site_secret":              "",
	"captcha_cap_instance_url":                   "",
	"captcha_cap_site_key":                       "",
	"captcha_cap_secret_key":                     "",
	"captcha_cap_asset_server":                   "jsdelivr",
	"thumb_width":                                "400",
	"thumb_height":                               "300",
	"thumb_entity_suffix":                        "._thumb",
	"thumb_slave_sidecar_suffix":                 "._thumb_sidecar",
	"thumb_encode_method":                        "png",
	"thumb_gc_after_gen":                         "0",
	"thumb_encode_quality":                       "95",
	"thumb_builtin_enabled":                      "1",
	"thumb_builtin_max_size":                     "78643200", // 75 MB
	"thumb_vips_max_size":                        "78643200", // 75 MB
	"thumb_vips_enabled":                         "0",
	"thumb_vips_exts":                            "3fr,ari,arw,bay,braw,crw,cr2,cr3,cap,data,dcs,dcr,dng,drf,eip,erf,fff,gpr,iiq,k25,kdc,mdc,mef,mos,mrw,nef,nrw,obm,orf,pef,ptx,pxn,r3d,raf,raw,rwl,rw2,rwz,sr2,srf,srw,tif,x3f,csv,mat,img,hdr,pbm,pgm,ppm,pfm,pnm,svg,svgz,j2k,jp2,jpt,j2c,jpc,gif,png,jpg,jpeg,jpe,webp,tif,tiff,fits,fit,fts,exr,jxl,pdf,heic,heif,avif,svs,vms,vmu,ndpi,scn,mrxs,svslide,bif,raw",
	"thumb_ffmpeg_enabled":                       "0",
	"thumb_vips_path":                            "vips",
	"thumb_ffmpeg_path":                          "ffmpeg",
	"thumb_ffmpeg_max_size":                      "10737418240", // 10 GB
	"thumb_ffmpeg_exts":                          "3g2,3gp,asf,asx,avi,divx,flv,m2ts,m2v,m4v,mkv,mov,mp4,mpeg,mpg,mts,mxf,ogv,rm,swf,webm,wmv",
	"thumb_ffmpeg_seek":                          "00:00:01.00",
	"thumb_libreoffice_path":                     "soffice",
	"thumb_libreoffice_max_size":                 "78643200", // 75 MB
	"thumb_libreoffice_enabled":                  "0",
	"thumb_libreoffice_exts":                     "txt,pdf,md,ods,ots,fods,uos,xlsx,xml,xls,xlt,dif,dbf,html,slk,csv,xlsm,docx,dotx,doc,dot,rtf,xlsm,xlst,xls,xlw,xlc,xlt,pptx,ppsx,potx,pomx,ppt,pps,ppm,pot,pom",
	"thumb_music_cover_enabled":                  "1",
	"thumb_music_cover_exts":                     "mp3,m4a,ogg,flac",
	"thumb_music_cover_max_size":                 "1073741824", // 1 GB
	"thumb_libraw_enabled":                       "0",
	"thumb_libraw_path":                          "simple_dcraw",
	"thumb_libraw_max_size":                      "78643200", // 75 MB
	"thumb_libraw_exts":                          "3fr,ari,arw,bay,braw,crw,cr2,cr3,cap,data,dcs,dcr,dng,drf,eip,erf,fff,gpr,iiq,k25,kdc,mdc,mef,mos,mrw,nef,nrw,obm,orf,pef,ptx,pxn,r3d,raf,raw,rwl,rw2,rwz,sr2,srf,srw,tif,x3f",
	"phone_required":                             "false",
	"phone_enabled":                              "false",
	"show_app_promotion":                         "1",
	"public_resource_maxage":                     "86400",
	"viewer_session_timeout":                     "36000",
	"hash_id_salt":                               util.RandStringRunes(64),
	"mail_activation_template":                   `[{"language":"en-US","title":"Activate your account","body":"<html lang=en xmlns=http://www.w3.org/1999/xhtml xmlns:o=urn:schemas-microsoft-com:office:office xmlns:v=urn:schemas-microsoft-com:vml><title></title><meta charset=UTF-8><meta content=\"text/html; charset=UTF-8\"http-equiv=Content-Type><!--[if !mso]>--><meta content=\"IE=edge\"http-equiv=X-UA-Compatible><!--<![endif]--><meta content=\"\"name=x-apple-disable-message-reformatting><meta content=\"target-densitydpi=device-dpi\"name=viewport><meta content=true name=HandheldFriendly><meta content=\"width=device-width\"name=viewport><meta content=\"telephone=no, date=no, address=no, email=no, url=no\"name=format-detection><style>table{border-collapse:separate;table-layout:fixed;mso-table-lspace:0;mso-table-rspace:0}table td{border-collapse:collapse}.ExternalClass{width:100%}.ExternalClass,.ExternalClass div,.ExternalClass font,.ExternalClass p,.ExternalClass span,.ExternalClass td{line-height:100%}a,body,h1,h2,h3,li,p{-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}html{-webkit-text-size-adjust:none!important}#innerTable,body{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}#innerTable img+div{display:none;display:none!important}img{Margin:0;padding:0;-ms-interpolation-mode:bicubic}a,h1,h2,h3,p{line-height:inherit;overflow-wrap:normal;white-space:normal;word-break:break-word}a{text-decoration:none}h1,h2,h3,p{min-width:100%!important;width:100%!important;max-width:100%!important;display:inline-block!important;border:0;padding:0;margin:0}a[x-apple-data-detectors]{color:inherit!important;text-decoration:none!important;font-size:inherit!important;font-family:inherit!important;font-weight:inherit!important;line-height:inherit!important}u+#body a{color:inherit;text-decoration:none;font-size:inherit;font-family:inherit;font-weight:inherit;line-height:inherit}a[href^=mailto],a[href^=sms],a[href^=tel]{color:inherit;text-decoration:none}</style><style>@media (min-width:481px){.hd{display:none!important}}</style><style>@media (max-width:480px){.hm{display:none!important}}</style><style>@media (max-width:480px){.t41,.t46{mso-line-height-alt:0!important;line-height:0!important;display:none!important}.t42{padding:40px!important}.t44{border-radius:0!important;width:480px!important}.t15,.t39,.t9{width:398px!important}.t32{text-align:left!important}.t25{display:revert!important}.t27,.t31{vertical-align:top!important;width:auto!important;max-width:100%!important}}</style><!--[if !mso]>--><link href=\"https://fonts.googleapis.com/css2?family=Montserrat:wght@700&family=Sofia+Sans:wght@700&family=Open+Sans:wght@400;500;600&display=swap\"rel=stylesheet><!--<![endif]--><!--[if mso]><xml><o:officedocumentsettings><o:allowpng><o:pixelsperinch>96</o:pixelsperinch></o:officedocumentsettings></xml><![endif]--><body class=t49 id=body style=min-width:100%;Margin:0;padding:0;background-color:#fff><div style=background-color:#fff class=t48><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100%><tr><td class=t47 style=font-size:0;line-height:0;mso-line-height-rule:exactly;background-color:#fff align=center valign=top><!--[if mso]><v:background xmlns:v=urn:schemas-microsoft-com:vml fill=true stroke=false><v:fill color=#FFFFFF></v:background><![endif]--><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100% id=innerTable><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t41>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t45 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"width=600><![endif]--><!--[if !mso]>--><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t43 style=width:100% width=100%><tr><td class=t42 style=\"padding:44px 42px 32px 42px\"><table cellpadding=0 cellspacing=0 role=presentation style=width:100%!important width=100%><tr><td align=left><table cellpadding=0 cellspacing=0 role=presentation class=t4 style=Margin-right:auto><tr><!--[if mso]><td class=t3 style=width:42px width=42><![endif]--><!--[if !mso]>--><td class=t3 style=width:100px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t2 style=width:100% width=100%><tr><td class=t1><div style=font-size:0><a href=\"{{ .CommonContext.SiteUrl }}\"><img alt=\"\"class=t0 height=100 src=\"{{ .CommonContext.Logo.Normal }}\"style=display:block;border:0;height:auto;width:100%;Margin:0;max-width:100%></a></div></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:22px;line-height:22px;font-size:1px;display:block class=t5>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t10 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t8 style=width:100% width=100%><tr><td class=t7 style=\"padding:0 0 18px 0\"><h1 class=t6 style=\"margin:0;Margin:0;font-family:Montserrat,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:28px;font-weight:700;font-style:normal;font-size:24px;text-decoration:none;text-transform:none;letter-spacing:-1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:1px\">Confirm your account</h1></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:18px;line-height:18px;font-size:1px;display:block class=t11>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t16 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t15 style=width:514px width=514><![endif]--><!--[if !mso]>--><td class=t15 style=width:514px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t14 style=width:100% width=100%><tr><td class=t13><p class=t12 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:25px;font-weight:400;font-style:normal;font-size:15px;text-decoration:none;text-transform:none;letter-spacing:-.1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:3px\">Please click the button below to confirm your email address and finish setting up your account. This link is valid for 24 hours.</table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:24px;line-height:24px;font-size:1px;display:block class=t18>  </div><tr><td align=left><a href=\"{{ .Url }}\"><table cellpadding=0 cellspacing=0 role=presentation class=t22 style=margin-right:auto><tr><!--[if mso]><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><![endif]--><!--[if !mso]>--><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t20 style=width:auto><tr><td class=t19 style=\"line-height:34px;mso-line-height-rule:exactly;mso-text-raise:5px;padding:0 23px 0 23px\"><span class=t17 style=\"display:block;margin:0;Margin:0;font-family:Sofia Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:34px;font-weight:700;font-style:normal;font-size:16px;text-decoration:none;text-transform:none;letter-spacing:-.2px;direction:ltr;color:#fff;mso-line-height-rule:exactly;mso-text-raise:5px\">Confirm</span></table></table></a><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:40px;line-height:40px;font-size:1px;display:block class=t36>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t40 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t38 style=width:100% width=100%><tr><td class=t37 style=\"padding:24px 0 0 0\"><div style=width:100%;text-align:left class=t35><div style=display:inline-block class=t34><table cellpadding=0 cellspacing=0 role=presentation class=t33 align=left valign=top><tr class=t32><td><td class=t27 valign=top><table cellpadding=0 cellspacing=0 role=presentation class=t26 style=width:auto width=100%><tr><td class=t24 style=background-color:#fff;line-height:20px;mso-line-height-rule:exactly;mso-text-raise:2px><span class=t23 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:600;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#222;mso-line-height-rule:exactly;mso-text-raise:2px\">{{ .CommonContext.SiteBasic.Name }}</span> <span class=t28 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:500;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#b4becc;mso-line-height-rule:exactly;mso-text-raise:2px;margin-left:8px\">This email is sent automatically.</span><td class=t25 style=width:20px width=20></table><td></table></div></div></table></table></table></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t46>  </div></table></table></div><div style=\"display:none;white-space:nowrap;font:15px courier;line-height:0\"class=gmail-fix>                                                           </div>"},{"language":"zh-CN","title":"激活你的账号","body":"<html lang=zh-CN xmlns=http://www.w3.org/1999/xhtml xmlns:o=urn:schemas-microsoft-com:office:office xmlns:v=urn:schemas-microsoft-com:vml><title></title><meta charset=UTF-8><meta content=\"text/html; charset=UTF-8\"http-equiv=Content-Type><!--[if !mso]>--><meta content=\"IE=edge\"http-equiv=X-UA-Compatible><!--<![endif]--><meta content=\"\"name=x-apple-disable-message-reformatting><meta content=\"target-densitydpi=device-dpi\"name=viewport><meta content=true name=HandheldFriendly><meta content=\"width=device-width\"name=viewport><meta content=\"telephone=no, date=no, address=no, email=no, url=no\"name=format-detection><style>table{border-collapse:separate;table-layout:fixed;mso-table-lspace:0;mso-table-rspace:0}table td{border-collapse:collapse}.ExternalClass{width:100%}.ExternalClass,.ExternalClass div,.ExternalClass font,.ExternalClass p,.ExternalClass span,.ExternalClass td{line-height:100%}a,body,h1,h2,h3,li,p{-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}html{-webkit-text-size-adjust:none!important}#innerTable,body{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}#innerTable img+div{display:none;display:none!important}img{Margin:0;padding:0;-ms-interpolation-mode:bicubic}a,h1,h2,h3,p{line-height:inherit;overflow-wrap:normal;white-space:normal;word-break:break-word}a{text-decoration:none}h1,h2,h3,p{min-width:100%!important;width:100%!important;max-width:100%!important;display:inline-block!important;border:0;padding:0;margin:0}a[x-apple-data-detectors]{color:inherit!important;text-decoration:none!important;font-size:inherit!important;font-family:inherit!important;font-weight:inherit!important;line-height:inherit!important}u+#body a{color:inherit;text-decoration:none;font-size:inherit;font-family:inherit;font-weight:inherit;line-height:inherit}a[href^=mailto],a[href^=sms],a[href^=tel]{color:inherit;text-decoration:none}</style><style>@media (min-width:481px){.hd{display:none!important}}</style><style>@media (max-width:480px){.hm{display:none!important}}</style><style>@media (max-width:480px){.t41,.t46{mso-line-height-alt:0!important;line-height:0!important;display:none!important}.t42{padding:40px!important}.t44{border-radius:0!important;width:480px!important}.t15,.t39,.t9{width:398px!important}.t32{text-align:left!important}.t25{display:revert!important}.t27,.t31{vertical-align:top!important;width:auto!important;max-width:100%!important}}</style><!--[if !mso]>--><link href=\"https://fonts.googleapis.com/css2?family=Montserrat:wght@700&family=Sofia+Sans:wght@700&family=Open+Sans:wght@400;500;600&display=swap\"rel=stylesheet><!--<![endif]--><!--[if mso]><xml><o:officedocumentsettings><o:allowpng><o:pixelsperinch>96</o:pixelsperinch></o:officedocumentsettings></xml><![endif]--><body class=t49 id=body style=min-width:100%;Margin:0;padding:0;background-color:#fff><div style=background-color:#fff class=t48><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100%><tr><td class=t47 style=font-size:0;line-height:0;mso-line-height-rule:exactly;background-color:#fff align=center valign=top><!--[if mso]><v:background xmlns:v=urn:schemas-microsoft-com:vml fill=true stroke=false><v:fill color=#FFFFFF></v:background><![endif]--><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100% id=innerTable><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t41>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t45 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"width=600><![endif]--><!--[if !mso]>--><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t43 style=width:100% width=100%><tr><td class=t42 style=\"padding:44px 42px 32px 42px\"><table cellpadding=0 cellspacing=0 role=presentation style=width:100%!important width=100%><tr><td align=left><table cellpadding=0 cellspacing=0 role=presentation class=t4 style=Margin-right:auto><tr><!--[if mso]><td class=t3 style=width:42px width=42><![endif]--><!--[if !mso]>--><td class=t3 style=width:100px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t2 style=width:100% width=100%><tr><td class=t1><div style=font-size:0><a href=\"{{ .CommonContext.SiteUrl }}\"><img alt=\"\"class=t0 height=100 src=\"{{ .CommonContext.Logo.Normal }}\"style=display:block;border:0;height:auto;width:100%;Margin:0;max-width:100%></a></div></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:22px;line-height:22px;font-size:1px;display:block class=t5>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t10 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t8 style=width:100% width=100%><tr><td class=t7 style=\"padding:0 0 18px 0\"><h1 class=t6 style=\"margin:0;Margin:0;font-family:Montserrat,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:28px;font-weight:700;font-style:normal;font-size:24px;text-decoration:none;text-transform:none;letter-spacing:-1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:1px\">激活你的账号</h1></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:18px;line-height:18px;font-size:1px;display:block class=t11>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t16 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t15 style=width:514px width=514><![endif]--><!--[if !mso]>--><td class=t15 style=width:514px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t14 style=width:100% width=100%><tr><td class=t13><p class=t12 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:25px;font-weight:400;font-style:normal;font-size:15px;text-decoration:none;text-transform:none;letter-spacing:-.1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:3px\">请点击下方按钮确认你的电子邮箱并完成账号注册，此链接有效期为 24 小时。</table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:24px;line-height:24px;font-size:1px;display:block class=t18>  </div><tr><td align=left><a href=\"{{ .Url }}\"><table cellpadding=0 cellspacing=0 role=presentation class=t22 style=margin-right:auto><tr><!--[if mso]><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><![endif]--><!--[if !mso]>--><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t20 style=width:auto><tr><td class=t19 style=\"line-height:34px;mso-line-height-rule:exactly;mso-text-raise:5px;padding:0 23px 0 23px\"><span class=t17 style=\"display:block;margin:0;Margin:0;font-family:Sofia Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:34px;font-weight:700;font-style:normal;font-size:16px;text-decoration:none;text-transform:none;letter-spacing:-.2px;direction:ltr;color:#fff;mso-line-height-rule:exactly;mso-text-raise:5px\">确认激活</span></table></table></a><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:40px;line-height:40px;font-size:1px;display:block class=t36>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t40 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t38 style=width:100% width=100%><tr><td class=t37 style=\"padding:24px 0 0 0\"><div style=width:100%;text-align:left class=t35><div style=display:inline-block class=t34><table cellpadding=0 cellspacing=0 role=presentation class=t33 align=left valign=top><tr class=t32><td><td class=t27 valign=top><table cellpadding=0 cellspacing=0 role=presentation class=t26 style=width:auto width=100%><tr><td class=t24 style=background-color:#fff;line-height:20px;mso-line-height-rule:exactly;mso-text-raise:2px><span class=t23 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:600;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#222;mso-line-height-rule:exactly;mso-text-raise:2px\">{{ .CommonContext.SiteBasic.Name }}</span> <span class=t28 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:500;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#b4becc;mso-line-height-rule:exactly;mso-text-raise:2px;margin-left:8px\">此邮件由系统自动发送。</span><td class=t25 style=width:20px width=20></table><td></table></div></div></table></table></table></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t46>  </div></table></table></div><div style=\"display:none;white-space:nowrap;font:15px courier;line-height:0\"class=gmail-fix>                                                           </div>"}]`,
	"mail_reset_template":                        `[{"language":"en-US","title":"Reset your password","body":"<html lang=en xmlns=http://www.w3.org/1999/xhtml xmlns:o=urn:schemas-microsoft-com:office:office xmlns:v=urn:schemas-microsoft-com:vml><title></title><meta charset=UTF-8><meta content=\"text/html; charset=UTF-8\"http-equiv=Content-Type><!--[if !mso]>--><meta content=\"IE=edge\"http-equiv=X-UA-Compatible><!--<![endif]--><meta content=\"\"name=x-apple-disable-message-reformatting><meta content=\"target-densitydpi=device-dpi\"name=viewport><meta content=true name=HandheldFriendly><meta content=\"width=device-width\"name=viewport><meta content=\"telephone=no, date=no, address=no, email=no, url=no\"name=format-detection><style>table{border-collapse:separate;table-layout:fixed;mso-table-lspace:0;mso-table-rspace:0}table td{border-collapse:collapse}.ExternalClass{width:100%}.ExternalClass,.ExternalClass div,.ExternalClass font,.ExternalClass p,.ExternalClass span,.ExternalClass td{line-height:100%}a,body,h1,h2,h3,li,p{-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}html{-webkit-text-size-adjust:none!important}#innerTable,body{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}#innerTable img+div{display:none;display:none!important}img{Margin:0;padding:0;-ms-interpolation-mode:bicubic}a,h1,h2,h3,p{line-height:inherit;overflow-wrap:normal;white-space:normal;word-break:break-word}a{text-decoration:none}h1,h2,h3,p{min-width:100%!important;width:100%!important;max-width:100%!important;display:inline-block!important;border:0;padding:0;margin:0}a[x-apple-data-detectors]{color:inherit!important;text-decoration:none!important;font-size:inherit!important;font-family:inherit!important;font-weight:inherit!important;line-height:inherit!important}u+#body a{color:inherit;text-decoration:none;font-size:inherit;font-family:inherit;font-weight:inherit;line-height:inherit}a[href^=mailto],a[href^=sms],a[href^=tel]{color:inherit;text-decoration:none}</style><style>@media (min-width:481px){.hd{display:none!important}}</style><style>@media (max-width:480px){.hm{display:none!important}}</style><style>@media (max-width:480px){.t41,.t46{mso-line-height-alt:0!important;line-height:0!important;display:none!important}.t42{padding:40px!important}.t44{border-radius:0!important;width:480px!important}.t15,.t39,.t9{width:398px!important}.t32{text-align:left!important}.t25{display:revert!important}.t27,.t31{vertical-align:top!important;width:auto!important;max-width:100%!important}}</style><!--[if !mso]>--><link href=\"https://fonts.googleapis.com/css2?family=Montserrat:wght@700&family=Sofia+Sans:wght@700&family=Open+Sans:wght@400;500;600&display=swap\"rel=stylesheet><!--<![endif]--><!--[if mso]><xml><o:officedocumentsettings><o:allowpng><o:pixelsperinch>96</o:pixelsperinch></o:officedocumentsettings></xml><![endif]--><body class=t49 id=body style=min-width:100%;Margin:0;padding:0;background-color:#fff><div style=background-color:#fff class=t48><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100%><tr><td class=t47 style=font-size:0;line-height:0;mso-line-height-rule:exactly;background-color:#fff align=center valign=top><!--[if mso]><v:background xmlns:v=urn:schemas-microsoft-com:vml fill=true stroke=false><v:fill color=#FFFFFF></v:background><![endif]--><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100% id=innerTable><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t41>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t45 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"width=600><![endif]--><!--[if !mso]>--><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t43 style=width:100% width=100%><tr><td class=t42 style=\"padding:44px 42px 32px 42px\"><table cellpadding=0 cellspacing=0 role=presentation style=width:100%!important width=100%><tr><td align=left><table cellpadding=0 cellspacing=0 role=presentation class=t4 style=Margin-right:auto><tr><!--[if mso]><td class=t3 style=width:42px width=42><![endif]--><!--[if !mso]>--><td class=t3 style=width:100px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t2 style=width:100% width=100%><tr><td class=t1><div style=font-size:0><a href=\"{{ .CommonContext.SiteUrl }}\"><img alt=\"\"class=t0 height=100 src=\"{{ .CommonContext.Logo.Normal }}\"style=display:block;border:0;height:auto;width:100%;Margin:0;max-width:100%></a></div></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:22px;line-height:22px;font-size:1px;display:block class=t5>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t10 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t8 style=width:100% width=100%><tr><td class=t7 style=\"padding:0 0 18px 0\"><h1 class=t6 style=\"margin:0;Margin:0;font-family:Montserrat,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:28px;font-weight:700;font-style:normal;font-size:24px;text-decoration:none;text-transform:none;letter-spacing:-1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:1px\">Reset your password</h1></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:18px;line-height:18px;font-size:1px;display:block class=t11>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t16 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t15 style=width:514px width=514><![endif]--><!--[if !mso]>--><td class=t15 style=width:514px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t14 style=width:100% width=100%><tr><td class=t13><p class=t12 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:25px;font-weight:400;font-style:normal;font-size:15px;text-decoration:none;text-transform:none;letter-spacing:-.1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:3px\">Please click the button below to reset your password. This link is valid for one hour.</table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:24px;line-height:24px;font-size:1px;display:block class=t18>  </div><tr><td align=left><a href=\"{{ .Url }}\"><table cellpadding=0 cellspacing=0 role=presentation class=t22 style=margin-right:auto><tr><!--[if mso]><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><![endif]--><!--[if !mso]>--><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t20 style=width:auto><tr><td class=t19 style=\"line-height:34px;mso-line-height-rule:exactly;mso-text-raise:5px;padding:0 23px 0 23px\"><span class=t17 style=\"display:block;margin:0;Margin:0;font-family:Sofia Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:34px;font-weight:700;font-style:normal;font-size:16px;text-decoration:none;text-transform:none;letter-spacing:-.2px;direction:ltr;color:#fff;mso-line-height-rule:exactly;mso-text-raise:5px\">Reset</span></table></table></a><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:40px;line-height:40px;font-size:1px;display:block class=t36>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t40 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t38 style=width:100% width=100%><tr><td class=t37 style=\"padding:24px 0 0 0\"><div style=width:100%;text-align:left class=t35><div style=display:inline-block class=t34><table cellpadding=0 cellspacing=0 role=presentation class=t33 align=left valign=top><tr class=t32><td><td class=t27 valign=top><table cellpadding=0 cellspacing=0 role=presentation class=t26 style=width:auto width=100%><tr><td class=t24 style=background-color:#fff;line-height:20px;mso-line-height-rule:exactly;mso-text-raise:2px><span class=t23 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:600;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#222;mso-line-height-rule:exactly;mso-text-raise:2px\">{{ .CommonContext.SiteBasic.Name }}</span> <span class=t28 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:500;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#b4becc;mso-line-height-rule:exactly;mso-text-raise:2px;margin-left:8px\">This email is sent automatically.</span><td class=t25 style=width:20px width=20></table><td></table></div></div></table></table></table></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t46>  </div></table></table></div><div style=\"display:none;white-space:nowrap;font:15px courier;line-height:0\"class=gmail-fix>                                                           </div>"},{"language":"zh-CN","title":"重设密码","body":"<html lang=zh-CN xmlns=http://www.w3.org/1999/xhtml xmlns:o=urn:schemas-microsoft-com:office:office xmlns:v=urn:schemas-microsoft-com:vml><title></title><meta charset=UTF-8><meta content=\"text/html; charset=UTF-8\"http-equiv=Content-Type><!--[if !mso]>--><meta content=\"IE=edge\"http-equiv=X-UA-Compatible><!--<![endif]--><meta content=\"\"name=x-apple-disable-message-reformatting><meta content=\"target-densitydpi=device-dpi\"name=viewport><meta content=true name=HandheldFriendly><meta content=\"width=device-width\"name=viewport><meta content=\"telephone=no, date=no, address=no, email=no, url=no\"name=format-detection><style>table{border-collapse:separate;table-layout:fixed;mso-table-lspace:0;mso-table-rspace:0}table td{border-collapse:collapse}.ExternalClass{width:100%}.ExternalClass,.ExternalClass div,.ExternalClass font,.ExternalClass p,.ExternalClass span,.ExternalClass td{line-height:100%}a,body,h1,h2,h3,li,p{-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}html{-webkit-text-size-adjust:none!important}#innerTable,body{-webkit-font-smoothing:antialiased;-moz-osx-font-smoothing:grayscale}#innerTable img+div{display:none;display:none!important}img{Margin:0;padding:0;-ms-interpolation-mode:bicubic}a,h1,h2,h3,p{line-height:inherit;overflow-wrap:normal;white-space:normal;word-break:break-word}a{text-decoration:none}h1,h2,h3,p{min-width:100%!important;width:100%!important;max-width:100%!important;display:inline-block!important;border:0;padding:0;margin:0}a[x-apple-data-detectors]{color:inherit!important;text-decoration:none!important;font-size:inherit!important;font-family:inherit!important;font-weight:inherit!important;line-height:inherit!important}u+#body a{color:inherit;text-decoration:none;font-size:inherit;font-family:inherit;font-weight:inherit;line-height:inherit}a[href^=mailto],a[href^=sms],a[href^=tel]{color:inherit;text-decoration:none}</style><style>@media (min-width:481px){.hd{display:none!important}}</style><style>@media (max-width:480px){.hm{display:none!important}}</style><style>@media (max-width:480px){.t41,.t46{mso-line-height-alt:0!important;line-height:0!important;display:none!important}.t42{padding:40px!important}.t44{border-radius:0!important;width:480px!important}.t15,.t39,.t9{width:398px!important}.t32{text-align:left!important}.t25{display:revert!important}.t27,.t31{vertical-align:top!important;width:auto!important;max-width:100%!important}}</style><!--[if !mso]>--><link href=\"https://fonts.googleapis.com/css2?family=Montserrat:wght@700&family=Sofia+Sans:wght@700&family=Open+Sans:wght@400;500;600&display=swap\"rel=stylesheet><!--<![endif]--><!--[if mso]><xml><o:officedocumentsettings><o:allowpng><o:pixelsperinch>96</o:pixelsperinch></o:officedocumentsettings></xml><![endif]--><body class=t49 id=body style=min-width:100%;Margin:0;padding:0;background-color:#fff><div style=background-color:#fff class=t48><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100%><tr><td class=t47 style=font-size:0;line-height:0;mso-line-height-rule:exactly;background-color:#fff align=center valign=top><!--[if mso]><v:background xmlns:v=urn:schemas-microsoft-com:vml fill=true stroke=false><v:fill color=#FFFFFF></v:background><![endif]--><table cellpadding=0 cellspacing=0 role=presentation align=center border=0 width=100% id=innerTable><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t41>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t45 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"width=600><![endif]--><!--[if !mso]>--><td class=t44 style=\"background-color:#fff;border:1px solid #ebebeb;overflow:hidden;width:600px;border-radius:12px 12px 12px 12px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t43 style=width:100% width=100%><tr><td class=t42 style=\"padding:44px 42px 32px 42px\"><table cellpadding=0 cellspacing=0 role=presentation style=width:100%!important width=100%><tr><td align=left><table cellpadding=0 cellspacing=0 role=presentation class=t4 style=Margin-right:auto><tr><!--[if mso]><td class=t3 style=width:42px width=42><![endif]--><!--[if !mso]>--><td class=t3 style=width:100px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t2 style=width:100% width=100%><tr><td class=t1><div style=font-size:0><a href=\"{{ .CommonContext.SiteUrl }}\"><img alt=\"\"class=t0 height=100 src=\"{{ .CommonContext.Logo.Normal }}\"style=display:block;border:0;height:auto;width:100%;Margin:0;max-width:100%></a></div></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:22px;line-height:22px;font-size:1px;display:block class=t5>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t10 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t9 style=\"border-bottom:1px solid #eff1f4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t8 style=width:100% width=100%><tr><td class=t7 style=\"padding:0 0 18px 0\"><h1 class=t6 style=\"margin:0;Margin:0;font-family:Montserrat,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:28px;font-weight:700;font-style:normal;font-size:24px;text-decoration:none;text-transform:none;letter-spacing:-1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:1px\">重设密码</h1></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:18px;line-height:18px;font-size:1px;display:block class=t11>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t16 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t15 style=width:514px width=514><![endif]--><!--[if !mso]>--><td class=t15 style=width:514px><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t14 style=width:100% width=100%><tr><td class=t13><p class=t12 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:25px;font-weight:400;font-style:normal;font-size:15px;text-decoration:none;text-transform:none;letter-spacing:-.1px;direction:ltr;color:#141414;text-align:left;mso-line-height-rule:exactly;mso-text-raise:3px\">请点击下方按钮重设你的密码，此链接有效期为 1 小时。</table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:24px;line-height:24px;font-size:1px;display:block class=t18>  </div><tr><td align=left><a href=\"{{ .Url }}\"><table cellpadding=0 cellspacing=0 role=presentation class=t22 style=margin-right:auto><tr><!--[if mso]><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><![endif]--><!--[if !mso]>--><td class=t21 style=\"background-color:#0666eb;overflow:hidden;width:auto;border-radius:40px 40px 40px 40px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t20 style=width:auto><tr><td class=t19 style=\"line-height:34px;mso-line-height-rule:exactly;mso-text-raise:5px;padding:0 23px 0 23px\"><span class=t17 style=\"display:block;margin:0;Margin:0;font-family:Sofia Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:34px;font-weight:700;font-style:normal;font-size:16px;text-decoration:none;text-transform:none;letter-spacing:-.2px;direction:ltr;color:#fff;mso-line-height-rule:exactly;mso-text-raise:5px\">重设密码</span></table></table></a><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:40px;line-height:40px;font-size:1px;display:block class=t36>  </div><tr><td align=center><table cellpadding=0 cellspacing=0 role=presentation class=t40 style=Margin-left:auto;Margin-right:auto><tr><!--[if mso]><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"width=514><![endif]--><!--[if !mso]>--><td class=t39 style=\"border-top:1px solid #dfe1e4;width:514px\"><!--<![endif]--><table cellpadding=0 cellspacing=0 role=presentation class=t38 style=width:100% width=100%><tr><td class=t37 style=\"padding:24px 0 0 0\"><div style=width:100%;text-align:left class=t35><div style=display:inline-block class=t34><table cellpadding=0 cellspacing=0 role=presentation class=t33 align=left valign=top><tr class=t32><td><td class=t27 valign=top><table cellpadding=0 cellspacing=0 role=presentation class=t26 style=width:auto width=100%><tr><td class=t24 style=background-color:#fff;line-height:20px;mso-line-height-rule:exactly;mso-text-raise:2px><span class=t23 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:600;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#222;mso-line-height-rule:exactly;mso-text-raise:2px\">{{ .CommonContext.SiteBasic.Name }}</span> <span class=t28 style=\"margin:0;Margin:0;font-family:Open Sans,BlinkMacSystemFont,Segoe UI,Helvetica Neue,Arial,sans-serif;line-height:20px;font-weight:500;font-style:normal;font-size:14px;text-decoration:none;direction:ltr;color:#b4becc;mso-line-height-rule:exactly;mso-text-raise:2px;margin-left:8px\">此邮件由系统自动发送。</span><td class=t25 style=width:20px width=20></table><td></table></div></div></table></table></table></table></table><tr><td><div style=mso-line-height-rule:exactly;mso-line-height-alt:50px;line-height:50px;font-size:1px;display:block class=t46>  </div></table></table></div><div style=\"display:none;white-space:nowrap;font:15px courier;line-height:0\"class=gmail-fix>                                                           </div>"}]`,
	"access_token_ttl":                           "3600",
	"refresh_token_ttl":                          "1209600", // 2 weeks
	"use_cursor_pagination":                      "1",
	"max_page_size":                              "2000",
	"max_recursive_searched_folder":              "65535",
	"max_batched_file":                           "3000",
	"queue_media_meta_worker_num":                "30",
	"queue_media_meta_max_execution":             "600",
	"queue_media_meta_backoff_factor":            "2",
	"queue_media_meta_backoff_max_duration":      "60",
	"queue_media_meta_max_retry":                 "1",
	"queue_media_meta_retry_delay":               "0",
	"queue_thumb_worker_num":                     "15",
	"queue_thumb_max_execution":                  "300",
	"queue_thumb_backoff_factor":                 "2",
	"queue_thumb_backoff_max_duration":           "60",
	"queue_thumb_max_retry":                      "0",
	"queue_thumb_retry_delay":                    "0",
	"queue_recycle_worker_num":                   "5",
	"queue_recycle_max_execution":                "900",
	"queue_recycle_backoff_factor":               "2",
	"queue_recycle_backoff_max_duration":         "60",
	"queue_recycle_max_retry":                    "0",
	"queue_recycle_retry_delay":                  "0",
	"queue_io_intense_worker_num":                "30",
	"queue_io_intense_max_execution":             "2592000",
	"queue_io_intense_backoff_factor":            "2",
	"queue_io_intense_backoff_max_duration":      "600",
	"queue_io_intense_max_retry":                 "5",
	"queue_io_intense_retry_delay":               "0",
	"queue_remote_download_worker_num":           "5",
	"queue_remote_download_max_execution":        "864000",
	"queue_remote_download_backoff_factor":       "2",
	"queue_remote_download_backoff_max_duration": "600",
	"queue_remote_download_max_retry":            "5",
	"queue_remote_download_retry_delay":          "0",
	"entity_url_default_ttl":                     "3600",
	"entity_url_cache_margin":                    "600",
	"media_meta":                                 "1",
	"media_meta_exif":                            "1",
	"media_meta_exif_size_local":                 "1073741824",
	"media_meta_exif_size_remote":                "104857600",
	"media_meta_exif_brute_force":                "1",
	"media_meta_music":                           "1",
	"media_meta_music_size_local":                "1073741824",
	"media_exif_music_size_remote":               "1073741824",
	"media_meta_ffprobe":                         "0",
	"media_meta_ffprobe_path":                    "ffprobe",
	"media_meta_ffprobe_size_local":              "0",
	"media_meta_ffprobe_size_remote":             "0",
	"site_logo":                                  "/static/img/logo.svg",
	"site_logo_light":                            "/static/img/logo_light.svg",
	"tos_url":                                    "https://cloudreve.org/privacy-policy",
	"privacy_policy_url":                         "https://cloudreve.org/privacy-policy",
	"explorer_category_image_query":              "type=file&case_folding&use_or&name=*.bmp&name=*.iff&name=*.png&name=*.gif&name=*.jpg&name=*.jpeg&name=*.psd&name=*.svg&name=*.webp&name=*.heif&name=*.heic&name=*.tiff&name=*.avif&name=*.3fr&name=*.ari&name=*.arw&name=*.bay&name=*.braw&name=*.crw&name=*.cr2&name=*.cr3&name=*.cap&name=*.dcs&name=*.dcr&name=*.dng&name=*.drf&name=*.eip&name=*.erf&name=*.fff&name=*.gpr&name=*.iiq&name=*.k25&name=*.kdc&name=*.mdc&name=*.mef&name=*.mos&name=*.mrw&name=*.nef&name=*.nrw&name=*.obm&name=*.orf&name=*.pef&name=*.ptx&name=*.pxn&name=*.r3d&name=*.raf&name=*.raw&name=*.rwl&name=*.rw2&name=*.rwz&name=*.sr2&name=*.srf&name=*.srw&name=*.tif&name=*.x3f",
	"explorer_category_video_query":              "type=file&case_folding&use_or&name=*.mp4&name=*.m3u8&name=*.flv&name=*.avi&name=*.wmv&name=*.mkv&name=*.rm&name=*.rmvb&name=*.mov&name=*.ogv",
	"explorer_category_audio_query":              "type=file&case_folding&use_or&name=*.mp3&name=*.flac&name=*.ape&name=*.wav&name=*.acc&name=*.ogg&name=*.m4a",
	"explorer_category_document_query":           "type=file&case_folding&use_or&name=*.pdf&name=*.doc&name=*.docx&name=*.ppt&name=*.pptx&name=*.xls&name=*.xlsx&name=*.csv&name=*.txt&name=*.md&name=*.pub",
	"use_sse_for_search":                         "0",
	"emojis":                                     `{"😀":["😀","😃","😄","😁","😆","😅","🤣","😂","🙂","🙃","🫠","😉","😊","😇","🥰","😍","🤩","😘","😗","😚","😙","🥲","😋","😛","😜","🤪","😝","🤑","🤗","🤭","🫢","🫣","🤫","🤔","🫡","🤐","🤨","😐","😑","😶","😶‍🌫️","😏","😒","🙄","😬","😮‍💨","🤥","😌","😔","😪","🤤","😴","😷","🤒","🤕","🤢","🤮","🤧","🥵","🥶","🥴","😵","😵‍💫","🤯","🤠","🥳","🥸","😎","🤓","🧐","😕","🫤","😟","🙁","😮","😯","😲","😳","🥺","🥹","😦","😧","😨","😰","😥","😢","😭","😱","😖","😣","😞","😓","😩","😫","🥱","😤","😡","😠","🤬","😈","👿","💀","☠️","💩","🤡","👹","👺","👻","👽","👾","🤖","😺","😸","😹","😻","😼","😽","🙀","😿","😾","🙈","🙉","🙊","💋","💌","💘","💝","💖","💗","💓","💞","💕","💟","💔","❤️‍🔥","❤️‍🩹","❤️","🧡","💛","💚","💙","💜","🤎","🖤","🤍","💯","💢","💥","💫","💦","💨","🕳️","💣","💬","👁️‍🗨️","🗨️","🗯️","💭","💤"],"👋":["👋","🤚","🖐️","✋","🖖","🫱","🫲","🫳","🫴","👌","🤌","🤏","✌️","🤞","🫰","🤟","🤘","🤙","👈","👉","👆","🖕","👇","☝️","🫵","👍","👎","✊","👊","🤛","🤜","👏","🙌","🫶","👐","🤲","🤝","🙏","✍️","💅","🤳","💪","🦾","🦿","🦵","🦶","👂","🦻","👃","🧠","🫀","🫁","🦷","🦴","👀","👁️","👅","👄","🫦","👶","🧒","👦","👧","🧑","👱","👨","🧔","🧔‍♂️","🧔‍♀️","👨‍🦰","👨‍🦱","👨‍🦳","👨‍🦲","👩","👩‍🦰","🧑‍🦰","👩‍🦱","🧑‍🦱","👩‍🦳","🧑‍🦳","👩‍🦲","🧑‍🦲","👱‍♀️","👱‍♂️","🧓","👴","👵","🙍","🙍‍♂️","🙍‍♀️","🙎","🙎‍♂️","🙎‍♀️","🙅","🙅‍♂️","🙅‍♀️","🙆","🙆‍♂️","🙆‍♀️","💁","💁‍♂️","💁‍♀️","🙋","🙋‍♂️","🙋‍♀️","🧏","🧏‍♂️","🧏‍♀️","🙇","🙇‍♂️","🙇‍♀️","🤦","🤦‍♂️","🤦‍♀️","🤷","🤷‍♂️","🤷‍♀️","🧑‍⚕️","👨‍⚕️","👩‍⚕️","🧑‍🎓","👨‍🎓","👩‍🎓","🧑‍🏫","👨‍🏫","👩‍🏫","🧑‍⚖️","👨‍⚖️","👩‍⚖️","🧑‍🌾","👨‍🌾","👩‍🌾","🧑‍🍳","👨‍🍳","👩‍🍳","🧑‍🔧","👨‍🔧","👩‍🔧","🧑‍🏭","👨‍🏭","👩‍🏭","🧑‍💼","👨‍💼","👩‍💼","🧑‍🔬","👨‍🔬","👩‍🔬","🧑‍💻","👨‍💻","👩‍💻","🧑‍🎤","👨‍🎤","👩‍🎤","🧑‍🎨","👨‍🎨","👩‍🎨","🧑‍✈️","👨‍✈️","👩‍✈️","🧑‍🚀","👨‍🚀","👩‍🚀","🧑‍🚒","👨‍🚒","👩‍🚒","👮","👮‍♂️","👮‍♀️","🕵️","🕵️‍♂️","🕵️‍♀️","💂","💂‍♂️","💂‍♀️","🥷","👷","👷‍♂️","👷‍♀️","🫅","🤴","👸","👳","👳‍♂️","👳‍♀️","👲","🧕","🤵","🤵‍♂️","🤵‍♀️","👰","👰‍♂️","👰‍♀️","🤰","🫃","🫄","🤱","👩‍🍼","👨‍🍼","🧑‍🍼","👼","🎅","🤶","🧑‍🎄","🦸","🦸‍♂️","🦸‍♀️","🦹","🦹‍♂️","🦹‍♀️","🧙","🧙‍♂️","🧙‍♀️","🧚","🧚‍♂️","🧚‍♀️","🧛","🧛‍♂️","🧛‍♀️","🧜","🧜‍♂️","🧜‍♀️","🧝","🧝‍♂️","🧝‍♀️","🧞","🧞‍♂️","🧞‍♀️","🧟","🧟‍♂️","🧟‍♀️","🧌","💆","💆‍♂️","💆‍♀️","💇","💇‍♂️","💇‍♀️","🚶","🚶‍♂️","🚶‍♀️","🧍","🧍‍♂️","🧍‍♀️","🧎","🧎‍♂️","🧎‍♀️","🧑‍🦯","👨‍🦯","👩‍🦯","🧑‍🦼","👨‍🦼","👩‍🦼","🧑‍🦽","👨‍🦽","👩‍🦽","🏃","🏃‍♂️","🏃‍♀️","💃","🕺","🕴️","👯","👯‍♂️","👯‍♀️","🧖","🧖‍♂️","🧖‍♀️","🧗","🧗‍♂️","🧗‍♀️","🤺","🏇","⛷️","🏂","🏌️","🏌️‍♂️","🏌️‍♀️","🏄","🏄‍♂️","🏄‍♀️","🚣","🚣‍♂️","🚣‍♀️","🏊","🏊‍♂️","🏊‍♀️","⛹️","⛹️‍♂️","⛹️‍♀️","🏋️","🏋️‍♂️","🏋️‍♀️","🚴","🚴‍♂️","🚴‍♀️","🚵","🚵‍♂️","🚵‍♀️","🤸","🤸‍♂️","🤸‍♀️","🤼","🤼‍♂️","🤼‍♀️","🤽","🤽‍♂️","🤽‍♀️","🤾","🤾‍♂️","🤾‍♀️","🤹","🤹‍♂️","🤹‍♀️","🧘","🧘‍♂️","🧘‍♀️","🛀","🛌","🧑‍🤝‍🧑","👭","👫","👬","💏","👩‍❤️‍💋‍👨","👨‍❤️‍💋‍👨","👩‍❤️‍💋‍👩","💑","👩‍❤️‍👨","👨‍❤️‍👨","👩‍❤️‍👩","👪","👨‍👩‍👦","👨‍👩‍👧","👨‍👩‍👧‍👦","👨‍👩‍👦‍👦","👨‍👩‍👧‍👧","👨‍👨‍👦","👨‍👨‍👧","👨‍👨‍👧‍👦","👨‍👨‍👦‍👦","👨‍👨‍👧‍👧","👩‍👩‍👦","👩‍👩‍👧","👩‍👩‍👧‍👦","👩‍👩‍👦‍👦","👩‍👩‍👧‍👧","👨‍👦","👨‍👦‍👦","👨‍👧","👨‍👧‍👦","👨‍👧‍👧","👩‍👦","👩‍👦‍👦","👩‍👧","👩‍👧‍👦","👩‍👧‍👧","🗣️","👤","👥","🫂","👣","🦰","🦱","🦳","🦲"],"🐵":["🐵","🐒","🦍","🦧","🐶","🐕","🦮","🐕‍🦺","🐩","🐺","🦊","🦝","🐱","🐈","🐈‍⬛","🦁","🐯","🐅","🐆","🐴","🐎","🦄","🦓","🦌","🦬","🐮","🐂","🐃","🐄","🐷","🐖","🐗","🐽","🐏","🐑","🐐","🐪","🐫","🦙","🦒","🐘","🦣","🦏","🦛","🐭","🐁","🐀","🐹","🐰","🐇","🐿️","🦫","🦔","🦇","🐻","🐻‍❄️","🐨","🐼","🦥","🦦","🦨","🦘","🦡","🐾","🦃","🐔","🐓","🐣","🐤","🐥","🐦","🐧","🕊️","🦅","🦆","🦢","🦉","🦤","🪶","🦩","🦚","🦜","🐸","🐊","🐢","🦎","🐍","🐲","🐉","🦕","🦖","🐳","🐋","🐬","🦭","🐟","🐠","🐡","🦈","🐙","🐚","🪸","🐌","🦋","🐛","🐜","🐝","🪲","🐞","🦗","🪳","🕷️","🕸️","🦂","🦟","🪰","🪱","🦠","💐","🌸","💮","🪷","🏵️","🌹","🥀","🌺","🌻","🌼","🌷","🌱","🪴","🌲","🌳","🌴","🌵","🌾","🌿","☘️","🍀","🍁","🍂","🍃","🪹","🪺"],"🍇":["🍇","🍈","🍉","🍊","🍋","🍌","🍍","🥭","🍎","🍏","🍐","🍑","🍒","🍓","🫐","🥝","🍅","🫒","🥥","🥑","🍆","🥔","🥕","🌽","🌶️","🫑","🥒","🥬","🥦","🧄","🧅","🍄","🥜","🫘","🌰","🍞","🥐","🥖","🫓","🥨","🥯","🥞","🧇","🧀","🍖","🍗","🥩","🥓","🍔","🍟","🍕","🌭","🥪","🌮","🌯","🫔","🥙","🧆","🥚","🍳","🥘","🍲","🫕","🥣","🥗","🍿","🧈","🧂","🥫","🍱","🍘","🍙","🍚","🍛","🍜","🍝","🍠","🍢","🍣","🍤","🍥","🥮","🍡","🥟","🥠","🥡","🦀","🦞","🦐","🦑","🦪","🍦","🍧","🍨","🍩","🍪","🎂","🍰","🧁","🥧","🍫","🍬","🍭","🍮","🍯","🍼","🥛","☕","🫖","🍵","🍶","🍾","🍷","🍸","🍹","🍺","🍻","🥂","🥃","🫗","🥤","🧋","🧃","🧉","🧊","🥢","🍽️","🍴","🥄","🔪","🫙","🏺"],"🌍":["🌍","🌎","🌏","🌐","🗺️","🗾","🧭","🏔️","⛰️","🌋","🗻","🏕️","🏖️","🏜️","🏝️","🏞️","🏟️","🏛️","🏗️","🧱","🪨","🪵","🛖","🏘️","🏚️","🏠","🏡","🏢","🏣","🏤","🏥","🏦","🏨","🏩","🏪","🏫","🏬","🏭","🏯","🏰","💒","🗼","🗽","⛪","🕌","🛕","🕍","⛩️","🕋","⛲","⛺","🌁","🌃","🏙️","🌄","🌅","🌆","🌇","🌉","♨️","🎠","🛝","🎡","🎢","💈","🎪","🚂","🚃","🚄","🚅","🚆","🚇","🚈","🚉","🚊","🚝","🚞","🚋","🚌","🚍","🚎","🚐","🚑","🚒","🚓","🚔","🚕","🚖","🚗","🚘","🚙","🛻","🚚","🚛","🚜","🏎️","🏍️","🛵","🦽","🦼","🛺","🚲","🛴","🛹","🛼","🚏","🛣️","🛤️","🛢️","⛽","🛞","🚨","🚥","🚦","🛑","🚧","⚓","🛟","⛵","🛶","🚤","🛳️","⛴️","🛥️","🚢","✈️","🛩️","🛫","🛬","🪂","💺","🚁","🚟","🚠","🚡","🛰️","🚀","🛸","🛎️","🧳","⌛","⏳","⌚","⏰","⏱️","⏲️","🕰️","🕛","🕧","🕐","🕜","🕑","🕝","🕒","🕞","🕓","🕟","🕔","🕠","🕕","🕡","🕖","🕢","🕗","🕣","🕘","🕤","🕙","🕥","🕚","🕦","🌑","🌒","🌓","🌔","🌕","🌖","🌗","🌘","🌙","🌚","🌛","🌜","🌡️","☀️","🌝","🌞","🪐","⭐","🌟","🌠","🌌","☁️","⛅","⛈️","🌤️","🌥️","🌦️","🌧️","🌨️","🌩️","🌪️","🌫️","🌬️","🌀","🌈","🌂","☂️","☔","⛱️","⚡","❄️","☃️","⛄","☄️","🔥","💧","🌊"],"🎃":["🎃","🎄","🎆","🎇","🧨","✨","🎈","🎉","🎊","🎋","🎍","🎎","🎏","🎐","🎑","🧧","🎀","🎁","🎗️","🎟️","🎫","🎖️","🏆","🏅","🥇","🥈","🥉","⚽","⚾","🥎","🏀","🏐","🏈","🏉","🎾","🥏","🎳","🏏","🏑","🏒","🥍","🏓","🏸","🥊","🥋","🥅","⛳","⛸️","🎣","🤿","🎽","🎿","🛷","🥌","🎯","🪀","🪁","🎱","🔮","🪄","🧿","🪬","🎮","🕹️","🎰","🎲","🧩","🧸","🪅","🪩","🪆","♠️","♥️","♦️","♣️","♟️","🃏","🀄","🎴","🎭","🖼️","🎨","🧵","🪡","🧶","🪢"],"👓":["👓","🕶️","🥽","🥼","🦺","👔","👕","👖","🧣","🧤","🧥","🧦","👗","👘","🥻","🩱","🩲","🩳","👙","👚","👛","👜","👝","🛍️","🎒","🩴","👞","👟","🥾","🥿","👠","👡","🩰","👢","👑","👒","🎩","🎓","🧢","🪖","⛑️","📿","💄","💍","💎","🔇","🔈","🔉","🔊","📢","📣","📯","🔔","🔕","🎼","🎵","🎶","🎙️","🎚️","🎛️","🎤","🎧","📻","🎷","🪗","🎸","🎹","🎺","🎻","🪕","🥁","🪘","📱","📲","☎️","📞","📟","📠","🔋","🪫","🔌","💻","🖥️","🖨️","⌨️","🖱️","🖲️","💽","💾","💿","📀","🧮","🎥","🎞️","📽️","🎬","📺","📷","📸","📹","📼","🔍","🔎","🕯️","💡","🔦","🏮","🪔","📔","📕","📖","📗","📘","📙","📚","📓","📒","📃","📜","📄","📰","🗞️","📑","🔖","🏷️","💰","🪙","💴","💵","💶","💷","💸","💳","🧾","💹","✉️","📧","📨","📩","📤","📥","📦","📫","📪","📬","📭","📮","🗳️","✏️","✒️","🖋️","🖊️","🖌️","🖍️","📝","💼","📁","📂","🗂️","📅","📆","🗒️","🗓️","📇","📈","📉","📊","📋","📌","📍","📎","🖇️","📏","📐","✂️","🗃️","🗄️","🗑️","🔒","🔓","🔏","🔐","🔑","🗝️","🔨","🪓","⛏️","⚒️","🛠️","🗡️","⚔️","🔫","🪃","🏹","🛡️","🪚","🔧","🪛","🔩","⚙️","🗜️","⚖️","🦯","🔗","⛓️","🪝","🧰","🧲","🪜","⚗️","🧪","🧫","🧬","🔬","🔭","📡","💉","🩸","💊","🩹","🩼","🩺","🩻","🚪","🛗","🪞","🪟","🛏️","🛋️","🪑","🚽","🪠","🚿","🛁","🪤","🪒","🧴","🧷","🧹","🧺","🧻","🪣","🧼","🫧","🪥","🧽","🧯","🛒","🚬","⚰️","🪦","⚱️","🗿","🪧","🪪"],"🏧":["🏧","🚮","🚰","♿","🚹","🚺","🚻","🚼","🚾","🛂","🛃","🛄","🛅","⚠️","🚸","⛔","🚫","🚳","🚭","🚯","🚱","🚷","📵","🔞","☢️","☣️","⬆️","↗️","➡️","↘️","⬇️","↙️","⬅️","↖️","↕️","↔️","↩️","↪️","⤴️","⤵️","🔃","🔄","🔙","🔚","🔛","🔜","🔝","🛐","⚛️","🕉️","✡️","☸️","☯️","✝️","☦️","☪️","☮️","🕎","🔯","♈","♉","♊","♋","♌","♍","♎","♏","♐","♑","♒","♓","⛎","🔀","🔁","🔂","▶️","⏩","⏭️","⏯️","◀️","⏪","⏮️","🔼","⏫","🔽","⏬","⏸️","⏹️","⏺️","⏏️","🎦","🔅","🔆","📶","📳","📴","♀️","♂️","⚧️","✖️","➕","➖","➗","🟰","♾️","‼️","⁉️","❓","❔","❕","❗","〰️","💱","💲","⚕️","♻️","⚜️","🔱","📛","🔰","⭕","✅","☑️","✔️","❌","❎","➰","➿","〽️","✳️","✴️","❇️","©️","®️","™️","#️⃣","*️⃣","0️⃣","1️⃣","2️⃣","3️⃣","4️⃣","5️⃣","6️⃣","7️⃣","8️⃣","9️⃣","🔟","🔠","🔡","🔢","🔣","🔤","🅰️","🆎","🅱️","🆑","🆒","🆓","ℹ️","🆔","Ⓜ️","🆕","🆖","🅾️","🆗","🅿️","🆘","🆙","🆚","🈁","🈂️","🈷️","🈶","🈯","🉐","🈹","🈚","🈲","🉑","🈸","🈴","🈳","㊗️","㊙️","🈺","🈵","🔴","🟠","🟡","🟢","🔵","🟣","🟤","⚫","⚪","🟥","🟧","🟨","🟩","🟦","🟪","🟫","⬛","⬜","◼️","◻️","◾","◽","▪️","▫️","🔶","🔷","🔸","🔹","🔺","🔻","💠","🔘","🔳","🔲"],"🏁":["🏁","🚩","🎌","🏴","🏳️","🏳️‍🌈","🏳️‍⚧️","🏴‍☠️","🇦🇨","🇦🇩","🇦🇪","🇦🇫","🇦🇬","🇦🇮","🇦🇱","🇦🇲","🇦🇴","🇦🇶","🇦🇷","🇦🇸","🇦🇹","🇦🇺","🇦🇼","🇦🇽","🇦🇿","🇧🇦","🇧🇧","🇧🇩","🇧🇪","🇧🇫","🇧🇬","🇧🇭","🇧🇮","🇧🇯","🇧🇱","🇧🇲","🇧🇳","🇧🇴","🇧🇶","🇧🇷","🇧🇸","🇧🇹","🇧🇻","🇧🇼","🇧🇾","🇧🇿","🇨🇦","🇨🇨","🇨🇩","🇨🇫","🇨🇬","🇨🇭","🇨🇮","🇨🇰","🇨🇱","🇨🇲","🇨🇳","🇨🇴","🇨🇵","🇨🇷","🇨🇺","🇨🇻","🇨🇼","🇨🇽","🇨🇾","🇨🇿","🇩🇪","🇩🇬","🇩🇯","🇩🇰","🇩🇲","🇩🇴","🇩🇿","🇪🇦","🇪🇨","🇪🇪","🇪🇬","🇪🇭","🇪🇷","🇪🇸","🇪🇹","🇪🇺","🇫🇮","🇫🇯","🇫🇰","🇫🇲","🇫🇴","🇫🇷","🇬🇦","🇬🇧","🇬🇩","🇬🇪","🇬🇫","🇬🇬","🇬🇭","🇬🇮","🇬🇱","🇬🇲","🇬🇳","🇬🇵","🇬🇶","🇬🇷","🇬🇸","🇬🇹","🇬🇺","🇬🇼","🇬🇾","🇭🇰","🇭🇲","🇭🇳","🇭🇷","🇭🇹","🇭🇺","🇮🇨","🇮🇩","🇮🇪","🇮🇱","🇮🇲","🇮🇳","🇮🇴","🇮🇶","🇮🇷","🇮🇸","🇮🇹","🇯🇪","🇯🇲","🇯🇴","🇯🇵","🇰🇪","🇰🇬","🇰🇭","🇰🇮","🇰🇲","🇰🇳","🇰🇵","🇰🇷","🇰🇼","🇰🇾","🇰🇿","🇱🇦","🇱🇧","🇱🇨","🇱🇮","🇱🇰","🇱🇷","🇱🇸","🇱🇹","🇱🇺","🇱🇻","🇱🇾","🇲🇦","🇲🇨","🇲🇩","🇲🇪","🇲🇫","🇲🇬","🇲🇭","🇲🇰","🇲🇱","🇲🇲","🇲🇳","🇲🇴","🇲🇵","🇲🇶","🇲🇷","🇲🇸","🇲🇹","🇲🇺","🇲🇻","🇲🇼","🇲🇽","🇲🇾","🇲🇿","🇳🇦","🇳🇨","🇳🇪","🇳🇫","🇳🇬","🇳🇮","🇳🇱","🇳🇴","🇳🇵","🇳🇷","🇳🇺","🇳🇿","🇴🇲","🇵🇦","🇵🇪","🇵🇫","🇵🇬","🇵🇭","🇵🇰","🇵🇱","🇵🇲","🇵🇳","🇵🇷","🇵🇸","🇵🇹","🇵🇼","🇵🇾","🇶🇦","🇷🇪","🇷🇴","🇷🇸","🇷🇺","🇷🇼","🇸🇦","🇸🇧","🇸🇨","🇸🇩","🇸🇪","🇸🇬","🇸🇭","🇸🇮","🇸🇯","🇸🇰","🇸🇱","🇸🇲","🇸🇳","🇸🇴","🇸🇷","🇸🇸","🇸🇹","🇸🇻","🇸🇽","🇸🇾","🇸🇿","🇹🇦","🇹🇨","🇹🇩","🇹🇫","🇹🇬","🇹🇭","🇹🇯","🇹🇰","🇹🇱","🇹🇲","🇹🇳","🇹🇴","🇹🇷","🇹🇹","🇹🇻","🇹🇼","🇹🇿","🇺🇦","🇺🇬","🇺🇲","🇺🇳","🇺🇸","🇺🇾","🇺🇿","🇻🇦","🇻🇨","🇻🇪","🇻🇬","🇻🇮","🇻🇳","🇻🇺","🇼🇫","🇼🇸","🇽🇰","🇾🇪","🇾🇹","🇿🇦","🇿🇲","🇿🇼","🏴󠁧󠁢󠁥󠁮󠁧󠁿","🏴󠁧󠁢󠁳󠁣󠁴󠁿","🏴󠁧󠁢󠁷󠁬󠁳󠁿"]}`,
	"map_provider":                               "openstreetmap",
	"map_google_tile_type":                       "regular",
	"mime_mapping":                               `{".xlsx":"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",".xltx":"application/vnd.openxmlformats-officedocument.spreadsheetml.template",".potx":"application/vnd.openxmlformats-officedocument.presentationml.template",".ppsx":"application/vnd.openxmlformats-officedocument.presentationml.slideshow",".pptx":"application/vnd.openxmlformats-officedocument.presentationml.presentation",".sldx":"application/vnd.openxmlformats-officedocument.presentationml.slide",".docx":"application/vnd.openxmlformats-officedocument.wordprocessingml.document",".dotx":"application/vnd.openxmlformats-officedocument.wordprocessingml.template",".xlam":"application/vnd.ms-excel.addin.macroEnabled.12",".xlsb":"application/vnd.ms-excel.sheet.binary.macroEnabled.12",".apk":"application/vnd.android.package-archive",".hqx":"application/mac-binhex40",".cpt":"application/mac-compactpro",".doc":"application/msword",".ogg":"application/ogg",".pdf":"application/pdf",".rtf":"text/rtf",".mif":"application/vnd.mif",".xls":"application/vnd.ms-excel",".ppt":"application/vnd.ms-powerpoint",".odc":"application/vnd.oasis.opendocument.chart",".odb":"application/vnd.oasis.opendocument.database",".odf":"application/vnd.oasis.opendocument.formula",".odg":"application/vnd.oasis.opendocument.graphics",".otg":"application/vnd.oasis.opendocument.graphics-template",".odi":"application/vnd.oasis.opendocument.image",".odp":"application/vnd.oasis.opendocument.presentation",".otp":"application/vnd.oasis.opendocument.presentation-template",".ods":"application/vnd.oasis.opendocument.spreadsheet",".ots":"application/vnd.oasis.opendocument.spreadsheet-template",".odt":"application/vnd.oasis.opendocument.text",".odm":"application/vnd.oasis.opendocument.text-master",".ott":"application/vnd.oasis.opendocument.text-template",".oth":"application/vnd.oasis.opendocument.text-web",".sxw":"application/vnd.sun.xml.writer",".stw":"application/vnd.sun.xml.writer.template",".sxc":"application/vnd.sun.xml.calc",".stc":"application/vnd.sun.xml.calc.template",".sxd":"application/vnd.sun.xml.draw",".std":"application/vnd.sun.xml.draw.template",".sxi":"application/vnd.sun.xml.impress",".sti":"application/vnd.sun.xml.impress.template",".sxg":"application/vnd.sun.xml.writer.global",".sxm":"application/vnd.sun.xml.math",".sis":"application/vnd.symbian.install",".wbxml":"application/vnd.wap.wbxml",".wmlc":"application/vnd.wap.wmlc",".wmlsc":"application/vnd.wap.wmlscriptc",".bcpio":"application/x-bcpio",".torrent":"application/x-bittorrent",".bz2":"application/x-bzip2",".vcd":"application/x-cdlink",".pgn":"application/x-chess-pgn",".cpio":"application/x-cpio",".csh":"application/x-csh",".dvi":"application/x-dvi",".spl":"application/x-futuresplash",".gtar":"application/x-gtar",".hdf":"application/x-hdf",".jar":"application/x-java-archive",".jnlp":"application/x-java-jnlp-file",".js":"application/x-javascript",".ksp":"application/x-kspread",".chrt":"application/x-kchart",".kil":"application/x-killustrator",".latex":"application/x-latex",".rpm":"application/x-rpm",".sh":"application/x-sh",".shar":"application/x-shar",".swf":"application/x-shockwave-flash",".sit":"application/x-stuffit",".sv4cpio":"application/x-sv4cpio",".sv4crc":"application/x-sv4crc",".tar":"application/x-tar",".tcl":"application/x-tcl",".tex":"application/x-tex",".man":"application/x-troff-man",".me":"application/x-troff-me",".ms":"application/x-troff-ms",".ustar":"application/x-ustar",".src":"application/x-wais-source",".zip":"application/zip",".m3u":"audio/x-mpegurl",".ra":"audio/x-pn-realaudio",".wav":"audio/x-wav",".wma":"audio/x-ms-wma",".wax":"audio/x-ms-wax",".pdb":"chemical/x-pdb",".xyz":"chemical/x-xyz",".bmp":"image/bmp",".gif":"image/gif",".ief":"image/ief",".png":"image/png",".wbmp":"image/vnd.wap.wbmp",".ras":"image/x-cmu-raster",".pnm":"image/x-portable-anymap",".pbm":"image/x-portable-bitmap",".pgm":"image/x-portable-graymap",".ppm":"image/x-portable-pixmap",".rgb":"image/x-rgb",".xbm":"image/x-xbitmap",".xpm":"image/x-xpixmap",".xwd":"image/x-xwindowdump",".css":"text/css",".rtx":"text/richtext",".tsv":"text/tab-separated-values",".jad":"text/vnd.sun.j2me.app-descriptor",".wml":"text/vnd.wap.wml",".wmls":"text/vnd.wap.wmlscript",".etx":"text/x-setext",".mxu":"video/vnd.mpegurl",".flv":"video/x-flv",".wm":"video/x-ms-wm",".wmv":"video/x-ms-wmv",".wmx":"video/x-ms-wmx",".wvx":"video/x-ms-wvx",".avi":"video/x-msvideo",".movie":"video/x-sgi-movie",".ice":"x-conference/x-cooltalk",".3gp":"video/3gpp",".ai":"application/postscript",".aif":"audio/x-aiff",".aifc":"audio/x-aiff",".aiff":"audio/x-aiff",".asc":"text/plain",".atom":"application/atom+xml",".au":"audio/basic",".bin":"application/octet-stream",".cdf":"application/x-netcdf",".cgm":"image/cgm",".class":"application/octet-stream",".dcr":"application/x-director",".dif":"video/x-dv",".dir":"application/x-director",".djv":"image/vnd.djvu",".djvu":"image/vnd.djvu",".dll":"application/octet-stream",".dmg":"application/octet-stream",".dms":"application/octet-stream",".dtd":"application/xml-dtd",".dv":"video/x-dv",".dxr":"application/x-director",".eps":"application/postscript",".exe":"application/octet-stream",".ez":"application/andrew-inset",".gram":"application/srgs",".grxml":"application/srgs+xml",".gz":"application/x-gzip",".htm":"text/html",".html":"text/html",".ico":"image/x-icon",".ics":"text/calendar",".ifb":"text/calendar",".iges":"model/iges",".igs":"model/iges",".jp2":"image/jp2",".jpe":"image/jpeg",".jpeg":"image/jpeg",".jpg":"image/jpeg",".kar":"audio/midi",".lha":"application/octet-stream",".lzh":"application/octet-stream",".m4a":"audio/mp4a-latm",".m4p":"audio/mp4a-latm",".m4u":"video/vnd.mpegurl",".m4v":"video/x-m4v",".mac":"image/x-macpaint",".mathml":"application/mathml+xml",".mesh":"model/mesh",".mid":"audio/midi",".midi":"audio/midi",".mov":"video/quicktime",".mp2":"audio/mpeg",".mp3":"audio/mpeg",".mp4":"video/mp4",".mpe":"video/mpeg",".mpeg":"video/mpeg",".mpg":"video/mpeg",".mpga":"audio/mpeg",".msh":"model/mesh",".nc":"application/x-netcdf",".oda":"application/oda",".ogv":"video/ogv",".pct":"image/pict",".pic":"image/pict",".pict":"image/pict",".pnt":"image/x-macpaint",".pntg":"image/x-macpaint",".ps":"application/postscript",".qt":"video/quicktime",".qti":"image/x-quicktime",".qtif":"image/x-quicktime",".ram":"audio/x-pn-realaudio",".rdf":"application/rdf+xml",".rm":"application/vnd.rn-realmedia",".roff":"application/x-troff",".sgm":"text/sgml",".sgml":"text/sgml",".silo":"model/mesh",".skd":"application/x-koan",".skm":"application/x-koan",".skp":"application/x-koan",".skt":"application/x-koan",".smi":"application/smil",".smil":"application/smil",".snd":"audio/basic",".so":"application/octet-stream",".svg":"image/svg+xml",".t":"application/x-troff",".texi":"application/x-texinfo",".texinfo":"application/x-texinfo",".tif":"image/tiff",".tiff":"image/tiff",".tr":"application/x-troff",".txt":"text/plain; charset=utf-8",".vrml":"model/vrml",".vxml":"application/voicexml+xml",".webm":"video/webm",".wrl":"model/vrml",".xht":"application/xhtml+xml",".xhtml":"application/xhtml+xml",".xml":"application/xml",".xsl":"application/xml",".xslt":"application/xslt+xml",".xul":"application/vnd.mozilla.xul+xml",".webp":"image/webp",".323":"text/h323",".aab":"application/x-authoware-bin",".aam":"application/x-authoware-map",".aas":"application/x-authoware-seg",".acx":"application/internet-property-stream",".als":"audio/X-Alpha5",".amc":"application/x-mpeg",".ani":"application/octet-stream",".asd":"application/astound",".asf":"video/x-ms-asf",".asn":"application/astound",".asp":"application/x-asap",".asr":"video/x-ms-asf",".asx":"video/x-ms-asf",".avb":"application/octet-stream",".awb":"audio/amr-wb",".axs":"application/olescript",".bas":"text/plain",".bin ":"application/octet-stream",".bld":"application/bld",".bld2":"application/bld2",".bpk":"application/octet-stream",".c":"text/plain",".cal":"image/x-cals",".cat":"application/vnd.ms-pkiseccat",".ccn":"application/x-cnc",".cco":"application/x-cocoa",".cer":"application/x-x509-ca-cert",".cgi":"magnus-internal/cgi",".chat":"application/x-chat",".clp":"application/x-msclip",".cmx":"image/x-cmx",".co":"application/x-cult3d-object",".cod":"image/cis-cod",".conf":"text/plain",".cpp":"text/plain",".crd":"application/x-mscardfile",".crl":"application/pkix-crl",".crt":"application/x-x509-ca-cert",".csm":"chemical/x-csml",".csml":"chemical/x-csml",".cur":"application/octet-stream",".dcm":"x-lml/x-evm",".dcx":"image/x-dcx",".der":"application/x-x509-ca-cert",".dhtml":"text/html",".dot":"application/msword",".dwf":"drawing/x-dwf",".dwg":"application/x-autocad",".dxf":"application/x-autocad",".ebk":"application/x-expandedbook",".emb":"chemical/x-embl-dl-nucleotide",".embl":"chemical/x-embl-dl-nucleotide",".epub":"application/epub+zip",".eri":"image/x-eri",".es":"audio/echospeech",".esl":"audio/echospeech",".etc":"application/x-earthtime",".evm":"x-lml/x-evm",".evy":"application/envoy",".fh4":"image/x-freehand",".fh5":"image/x-freehand",".fhc":"image/x-freehand",".fif":"application/fractals",".flr":"x-world/x-vrml",".fm":"application/x-maker",".fpx":"image/x-fpx",".fvi":"video/isivideo",".gau":"chemical/x-gaussian-input",".gca":"application/x-gca-compressed",".gdb":"x-lml/x-gdb",".gps":"application/x-gps",".h":"text/plain",".hdm":"text/x-hdml",".hdml":"text/x-hdml",".hlp":"application/winhlp",".hta":"application/hta",".htc":"text/x-component",".hts":"text/html",".htt":"text/webviewhtml",".ifm":"image/gif",".ifs":"image/ifs",".iii":"application/x-iphone",".imy":"audio/melody",".ins":"application/x-internet-signup",".ips":"application/x-ipscript",".ipx":"application/x-ipix",".isp":"application/x-internet-signup",".it":"audio/x-mod",".itz":"audio/x-mod",".ivr":"i-world/i-vrml",".j2k":"image/j2k",".jam":"application/x-jam",".java":"text/plain",".jfif":"image/pipeg",".jpz":"image/jpeg",".jwc":"application/jwc",".kjx":"application/x-kjx",".lak":"x-lml/x-lak",".lcc":"application/fastman",".lcl":"application/x-digitalloca",".lcr":"application/x-digitalloca",".lgh":"application/lgh",".lml":"x-lml/x-lml",".lmlpack":"x-lml/x-lmlpack",".log":"text/plain",".lsf":"video/x-la-asf",".lsx":"video/x-la-asf",".m13":"application/x-msmediaview",".m14":"application/x-msmediaview",".m15":"audio/x-mod",".m3url":"audio/x-mpegurl",".m4b":"audio/mp4a-latm",".ma1":"audio/ma1",".ma2":"audio/ma2",".ma3":"audio/ma3",".ma5":"audio/ma5",".map":"magnus-internal/imagemap",".mbd":"application/mbedlet",".mct":"application/x-mascot",".mdb":"application/x-msaccess",".mdz":"audio/x-mod",".mel":"text/x-vmel",".mht":"message/rfc822",".mhtml":"message/rfc822",".mi":"application/x-mif",".mil":"image/x-cals",".mio":"audio/x-mio",".mmf":"application/x-skt-lbs",".mng":"video/x-mng",".mny":"application/x-msmoney",".moc":"application/x-mocha",".mocha":"application/x-mocha",".mod":"audio/x-mod",".mof":"application/x-yumekara",".mol":"chemical/x-mdl-molfile",".mop":"chemical/x-mopac-input",".mpa":"video/mpeg",".mpc":"application/vnd.mpohun.certificate",".mpg4":"video/mp4",".mpn":"application/vnd.mophun.application",".mpp":"application/vnd.ms-project",".mps":"application/x-mapserver",".mpv2":"video/mpeg",".mrl":"text/x-mrml",".mrm":"application/x-mrm",".msg":"application/vnd.ms-outlook",".mts":"application/metastream",".mtx":"application/metastream",".mtz":"application/metastream",".mvb":"application/x-msmediaview",".mzv":"application/metastream",".nar":"application/zip",".nbmp":"image/nbmp",".ndb":"x-lml/x-ndb",".ndwn":"application/ndwn",".nif":"application/x-nif",".nmz":"application/x-scream",".nokia-op-logo":"image/vnd.nok-oplogo-color",".npx":"application/x-netfpx",".nsnd":"audio/nsnd",".nva":"application/x-neva1",".nws":"message/rfc822",".oom":"application/x-AtlasMate-Plugin",".p10":"application/pkcs10",".p12":"application/x-pkcs12",".p7b":"application/x-pkcs7-certificates",".p7c":"application/x-pkcs7-mime",".p7m":"application/x-pkcs7-mime",".p7r":"application/x-pkcs7-certreqresp",".p7s":"application/x-pkcs7-signature",".pac":"audio/x-pac",".pae":"audio/x-epac",".pan":"application/x-pan",".pcx":"image/x-pcx",".pda":"image/x-pda",".pfr":"application/font-tdpfr",".pfx":"application/x-pkcs12",".pko":"application/ynd.ms-pkipko",".pm":"application/x-perl",".pma":"application/x-perfmon",".pmc":"application/x-perfmon",".pmd":"application/x-pmd",".pml":"application/x-perfmon",".pmr":"application/x-perfmon",".pmw":"application/x-perfmon",".pnz":"image/png",".pot,":"application/vnd.ms-powerpoint",".pps":"application/vnd.ms-powerpoint",".pqf":"application/x-cprplayer",".pqi":"application/cprplayer",".prc":"application/x-prc",".prf":"application/pics-rules",".prop":"text/plain",".proxy":"application/x-ns-proxy-autoconfig",".ptlk":"application/listenup",".pub":"application/x-mspublisher",".pvx":"video/x-pv-pvx",".qcp":"audio/vnd.qcelp",".r3t":"text/vnd.rn-realtext3d",".rar":"application/octet-stream",".rc":"text/plain",".rf":"image/vnd.rn-realflash",".rlf":"application/x-richlink",".rmf":"audio/x-rmf",".rmi":"audio/mid",".rmm":"audio/x-pn-realaudio",".rmvb":"audio/x-pn-realaudio",".rnx":"application/vnd.rn-realplayer",".rp":"image/vnd.rn-realpix",".rt":"text/vnd.rn-realtext",".rte":"x-lml/x-gps",".rtg":"application/metastream",".rv":"video/vnd.rn-realvideo",".rwc":"application/x-rogerwilco",".s3m":"audio/x-mod",".s3z":"audio/x-mod",".sca":"application/x-supercard",".scd":"application/x-msschedule",".sct":"text/scriptlet",".sdf":"application/e-score",".sea":"application/x-stuffit",".setpay":"application/set-payment_old-initiation",".setreg":"application/set-registration-initiation",".shtml":"text/html",".shtm":"text/html",".shw":"application/presentations",".si6":"image/si6",".si7":"image/vnd.stiwap.sis",".si9":"image/vnd.lgtwap.sis",".slc":"application/x-salsa",".smd":"audio/x-smd",".smp":"application/studiom",".smz":"audio/x-smd",".spc":"application/x-pkcs7-certificates",".spr":"application/x-sprite",".sprite":"application/x-sprite",".sdp":"application/sdp",".spt":"application/x-spt",".sst":"application/vnd.ms-pkicertstore",".stk":"application/hyperstudio",".stl":"application/vnd.ms-pkistl",".stm":"text/html",".svf":"image/vnd",".svh":"image/svh",".svr":"x-world/x-svr",".swfl":"application/x-shockwave-flash",".tad":"application/octet-stream",".talk":"text/x-speech",".taz":"application/x-tar",".tbp":"application/x-timbuktu",".tbt":"application/x-timbuktu",".tgz":"application/x-compressed",".thm":"application/vnd.eri.thm",".tki":"application/x-tkined",".tkined":"application/x-tkined",".toc":"application/toc",".toy":"image/toy",".trk":"x-lml/x-gps",".trm":"application/x-msterminal",".tsi":"audio/tsplayer",".tsp":"application/dsptype",".ttf":"application/octet-stream",".ttz":"application/t-time",".uls":"text/iuls",".ult":"audio/x-mod",".uu":"application/x-uuencode",".uue":"application/x-uuencode",".vcf":"text/x-vcard",".vdo":"video/vdo",".vib":"audio/vib",".viv":"video/vivo",".vivo":"video/vivo",".vmd":"application/vocaltec-media-desc",".vmf":"application/vocaltec-media-file",".vmi":"application/x-dreamcast-vms-info",".vms":"application/x-dreamcast-vms",".vox":"audio/voxware",".vqe":"audio/x-twinvq-plugin",".vqf":"audio/x-twinvq",".vql":"audio/x-twinvq",".vre":"x-world/x-vream",".vrt":"x-world/x-vrt",".vrw":"x-world/x-vream",".vts":"workbook/formulaone",".wcm":"application/vnd.ms-works",".wdb":"application/vnd.ms-works",".web":"application/vnd.xara",".wi":"image/wavelet",".wis":"application/x-InstallShield",".wks":"application/vnd.ms-works",".wmd":"application/x-ms-wmd",".wmf":"application/x-msmetafile",".wmlscript":"text/vnd.wap.wmlscript",".wmz":"application/x-ms-wmz",".wpng":"image/x-up-wpng",".wps":"application/vnd.ms-works",".wpt":"x-lml/x-gps",".wri":"application/x-mswrite",".wrz":"x-world/x-vrml",".ws":"text/vnd.wap.wmlscript",".wsc":"application/vnd.wap.wmlscriptc",".wv":"video/wavelet",".wxl":"application/x-wxl",".x-gzip":"application/x-gzip",".xaf":"x-world/x-vrml",".xar":"application/vnd.xara",".xdm":"application/x-xdma",".xdma":"application/x-xdma",".xdw":"application/vnd.fujixerox.docuworks",".xhtm":"application/xhtml+xml",".xla":"application/vnd.ms-excel",".xlc":"application/vnd.ms-excel",".xll":"application/x-excel",".xlm":"application/vnd.ms-excel",".xlt":"application/vnd.ms-excel",".xlw":"application/vnd.ms-excel",".xm":"audio/x-mod",".xmz":"audio/x-mod",".xof":"x-world/x-vrml",".xpi":"application/x-xpinstall",".xsit":"text/xml",".yz1":"application/x-yz1",".z":"application/x-compress",".zac":"application/x-zaurus-zac",".json":"application/json"}`,
	"logto_enabled":                              "0",
	"logto_config":                               `{"direct_sign_in":true,"display_name":"vas.sso"}`,
	"qq_login":                                   `0`,
	"qq_login_config":                            `{"direct_sign_in":false}`,
	"license":                                    "",
}

func init() {
	explorerIcons, err := json.Marshal(defaultIcons)
	if err != nil {
		panic(err)
	}
	DefaultSettings["explorer_icons"] = string(explorerIcons)

	viewers, err := json.Marshal(defaultFileViewers)
	if err != nil {
		panic(err)
	}

	DefaultSettings["file_viewers"] = string(viewers)
}
