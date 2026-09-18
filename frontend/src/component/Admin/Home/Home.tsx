import Giscus from "@giscus/react";
import { GitHub } from "@mui/icons-material";
import {
  Avatar,
  Box,
  Container,
  Divider,
  List,
  ListItem,
  ListItemAvatar,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Paper,
  Skeleton,
  styled,
  Typography,
} from "@mui/material";
import { blue, green, red, yellow } from "@mui/material/colors";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import dayjs from "dayjs";
import i18next from "i18next";
import { useCallback, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { CartesianGrid, Legend, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";
import { getDashboardSummary } from "../../../api/api.ts";
import { HomepageSummary } from "../../../api/dashboard.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import FacebookCircularProgress from "../../Common/CircularProgress.tsx";
import { SecondaryButton, SquareChip } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";
import Book from "../../Icons/Book.tsx";
import BoxMultipleFilled from "../../Icons/BoxMultipleFilled.tsx";
import Discord from "../../Icons/Discord.tsx";
import DocumentCopyFilled from "../../Icons/DocumentCopyFilled.tsx";
import HomeIcon from "../../Icons/Home.tsx";
import OpenFilled from "../../Icons/OpenFilled.tsx";
import PeopleFilled from "../../Icons/PeopleFilled.tsx";
import ShareFilled from "../../Icons/ShareFilled.tsx";
import SparkleFilled from "../../Icons/SparkleFilled.tsx";
import Telegram from "../../Icons/Telegram.tsx";
import PageContainer from "../../Pages/PageContainer.tsx";
import PageHeader from "../../Pages/PageHeader.tsx";
import ProDialog from "../Common/ProDialog.tsx";
import SiteUrlWarning from "./SiteUrlWarning.tsx";
import CommentMultiple from "../../Icons/CommentMultiple.tsx";

const StyledPaper = styled(Paper)(({ theme }) => ({
  padding: theme.spacing(3),
  boxShadow: "initial",
  border: "1px solid " + theme.palette.divider,
}));

const StyledListItemIcon = styled(ListItemIcon)(() => ({
  minWidth: 0,
}));

const Home = () => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [summary, setSummary] = useState<HomepageSummary | undefined>();
  const [chartLoading, setChartLoading] = useState(false);
  const [siteUrlWarning, setSiteUrlWarning] = useState(false);
  const [proDialogOpen, setProDialogOpen] = useState(false);
  useEffect(() => {
    loadSummary(false);
  }, []);

  const loadSummary = useCallback((loadChart?: boolean) => {
    if (loadChart) {
      setChartLoading(true);
    }
    dispatch(getDashboardSummary(loadChart))
      .then((r) => {
        setSummary(r);
        if (!loadChart) {
          const target = r.site_urls.find((site) => site == window.location.origin);
          if (!target) {
            setSiteUrlWarning(true);
          }
        }
      })
      .finally(() => {
        setChartLoading(false);
      });
  }, []);

  return (
    <PageContainer>
      <ProDialog open={proDialogOpen} onClose={() => setProDialogOpen(false)} />
      <SiteUrlWarning
        open={siteUrlWarning}
        onClose={() => setSiteUrlWarning(false)}
        existingUrls={summary?.site_urls ?? []}
      />
      <Container maxWidth="xl">
        <PageHeader title={t("nav.summary")} />
        <Grid container spacing={3}>
          <Grid alignContent={"stretch"} item xs={12} md={8} lg={9}>
            <StyledPaper>
              <Typography
                variant={"subtitle1"}
                fontWeight={500}
                sx={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                }}
              >
                {t("summary.trend")}
                <Typography variant={"body2"} color={"text.secondary"}>
                  {summary?.metrics_summary?.generated_at && (
                    <Trans
                      i18nKey={"summary.generatedAt"}
                      ns={"dashboard"}
                      components={[<TimeBadge datetime={summary?.metrics_summary?.generated_at} variant={"inherit"} />]}
                    />
                  )}
                </Typography>
              </Typography>
              <Divider sx={{ mb: 2, mt: 1 }} />
              <SwitchTransition>
                <CSSTransition
                  addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
                  classNames="fade"
                  key={`${!!summary?.metrics_summary}-${!!chartLoading}`}
                >
                  <Box>
                    {summary?.metrics_summary && (
                      <ResponsiveContainer width="100%" height={300}>
                        <LineChart
                          height={350}
                          data={summary?.metrics_summary.dates.map((i, d) => ({
                            name: dayjs(i).format("MM-DD"),
                            user: summary?.metrics_summary?.users[d] ?? 0,
                            file: summary?.metrics_summary?.files[d] ?? 0,
                            share: summary?.metrics_summary?.shares[d] ?? 0,
                          }))}
                        >
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" />
                          <YAxis
                            allowDecimals={false}
                            width={(() => {
                              const yAxisValue = [
                                ...(summary?.metrics_summary?.users ?? []),
                                ...(summary?.metrics_summary?.files ?? []),
                                ...(summary?.metrics_summary?.shares ?? []),
                              ];
                              const yAxisUpperLimit = yAxisValue.length ? Math.max(...yAxisValue) / 0.8 - 1 : 0;
                              const yAxisDigits = yAxisUpperLimit > 1 ? Math.floor(Math.log10(yAxisUpperLimit)) + 1 : 1;
                              return 3 + yAxisDigits * 9;
                            })()}
                          />
                          <Tooltip />
                          <Legend />
                          <Line name={t("nav.users")} type="monotone" dataKey="user" stroke={blue[600]} />
                          <Line name={t("nav.files")} type="monotone" dataKey="file" stroke={yellow[800]} />
                          <Line name={t("nav.shares")} type="monotone" dataKey="share" stroke={green[800]} />
                        </LineChart>
                      </ResponsiveContainer>
                    )}
                    {chartLoading && (
                      <Box
                        sx={{
                          height: "300px",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                        }}
                      >
                        <FacebookCircularProgress />
                      </Box>
                    )}
                    {!summary?.metrics_summary?.generated_at && !chartLoading && (
                      <Box
                        sx={{
                          height: "300px",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                        }}
                      >
                        <SecondaryButton onClick={() => loadSummary(true)}>
                          {t("application:fileManager.calculate")}
                        </SecondaryButton>
                      </Box>
                    )}
                  </Box>
                </CSSTransition>
              </SwitchTransition>
            </StyledPaper>
          </Grid>
          <Grid item xs={12} md={4} lg={3}>
            <StyledPaper>
              <Typography variant={"subtitle1"} fontWeight={500}>
                {t("summary.summary")}
              </Typography>
              <Divider sx={{ mb: 2, mt: 1 }} />
              <SwitchTransition>
                <CSSTransition
                  addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
                  classNames="fade"
                  key={`${!!summary?.metrics_summary}-${chartLoading}`}
                >
                  <Box>
                    {summary?.metrics_summary && (
                      <List disablePadding sx={{ minHeight: "300px" }}>
                        <ListItem>
                          <ListItemAvatar>
                            <Avatar
                              sx={{
                                backgroundColor: blue[100],
                                color: blue[600],
                              }}
                            >
                              <PeopleFilled />
                            </Avatar>
                          </ListItemAvatar>
                          <ListItemText
                            secondary={t("summary.totalUsers")}
                            primary={summary.metrics_summary.user_total.toLocaleString()}
                          />
                        </ListItem>
                        <ListItem>
                          <ListItemAvatar>
                            <Avatar
                              sx={{
                                backgroundColor: yellow[100],
                                color: yellow[800],
                              }}
                            >
                              <DocumentCopyFilled />
                            </Avatar>
                          </ListItemAvatar>
                          <ListItemText
                            secondary={t("summary.totalFilesAndFolders")}
                            primary={summary.metrics_summary.file_total.toLocaleString()}
                          />
                        </ListItem>
                        <ListItem>
                          <ListItemAvatar>
                            <Avatar
                              sx={{
                                backgroundColor: green[100],
                                color: green[800],
                              }}
                            >
                              <ShareFilled />
                            </Avatar>
                          </ListItemAvatar>
                          <ListItemText
                            secondary={t("summary.shareLinks")}
                            primary={summary.metrics_summary.share_total.toLocaleString()}
                          />
                        </ListItem>
                        <ListItem>
                          <ListItemAvatar>
                            <Avatar
                              sx={{
                                backgroundColor: red[100],
                                color: red[800],
                              }}
                            >
                              <BoxMultipleFilled />
                            </Avatar>
                          </ListItemAvatar>
                          <ListItemText
                            secondary={t("summary.totalBlobs")}
                            primary={summary.metrics_summary.entities_total.toLocaleString()}
                          />
                        </ListItem>
                      </List>
                    )}
                    {chartLoading && (
                      <Box
                        sx={{
                          height: "300px",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                        }}
                      >
                        <FacebookCircularProgress />
                      </Box>
                    )}
                    {!summary?.metrics_summary?.generated_at && !chartLoading && (
                      <Box
                        sx={{
                          height: "300px",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                        }}
                      >
                        <SecondaryButton onClick={() => loadSummary(true)}>
                          {t("application:fileManager.calculate")}
                        </SecondaryButton>
                      </Box>
                    )}
                  </Box>
                </CSSTransition>
              </SwitchTransition>
            </StyledPaper>
          </Grid>
          <Grid item xs={12} md={5} lg={4}>
            <StyledPaper sx={{ p: 0 }}>
              <Box sx={{ p: 3, display: "flex", alignItems: "center" }}>
                <Box component={"img"} sx={{ width: 70 }} alt="cloudreve" src={"/static/img/cloudreve.svg"} />
                <Box sx={{ ml: 2 }}>
                  <Typography variant={"h5"} fontWeight={600}>
                    Cloudreve
                    {summary && summary.version.pro && (
                      <SquareChip sx={{ ml: 1, height: "initial" }} size={"small"} color={"primary"} label={"Pro"} />
                    )}
                  </Typography>
                  <Typography variant={"subtitle2"} color={"text.secondary"}>
                    {summary ? summary.version.version : <Skeleton variant={"text"} width={70} />}
                    {summary && (
                      <Box component={"span"} sx={{ ml: 1, color: (t) => t.palette.action.disabled }}>
                        #{summary.version.commit}
                      </Box>
                    )}
                  </Typography>
                </Box>
              </Box>
              <Divider />
              <List component="nav" aria-label="main mailbox folders" sx={{ mx: 2 }}>
                <ListItemButton onClick={() => window.open("https://cloudreve.org")}>
                  <ListItemIcon>
                    <HomeIcon />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.homepage")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                <ListItemButton onClick={() => window.open("https://github.com/cloudreve/cloudreve")}>
                  <ListItemIcon>
                    <GitHub />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.github")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                <ListItemButton onClick={() => window.open("https://docs.cloudreve.org/")}>
                  <ListItemIcon>
                    <Book />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.documents")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                <ListItemButton onClick={() => window.open("https://discord.gg/WTpMFpZT76")}>
                  <ListItemIcon>
                    <Discord />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.discordCommunity")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                <ListItemButton onClick={() => window.open("https://t.me/cloudreve_official")}>
                  <ListItemIcon>
                    <Telegram />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.telegram")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                <ListItemButton onClick={() => window.open("https://github.com/cloudreve/cloudreve/discussions")}>
                  <ListItemIcon>
                    <CommentMultiple />
                  </ListItemIcon>
                  <ListItemText primary={t("summary.forum")} />
                  <StyledListItemIcon>
                    <OpenFilled />
                  </StyledListItemIcon>
                </ListItemButton>
                {summary && !summary.version.pro && (
                  <ListItemButton onClick={() => setProDialogOpen(true)}>
                    <ListItemIcon>
                      <SparkleFilled color={"primary"} />
                    </ListItemIcon>
                    <ListItemText primary={t("summary.buyPro")} />
                  </ListItemButton>
                )}
              </List>
              <Divider />
            </StyledPaper>
          </Grid>
          <Grid item xs={12} md={7} lg={8}>
            <StyledPaper>
              <Typography variant={"subtitle1"} fontWeight={500}>
                公告
              </Typography>
              <Divider sx={{ mb: 2, mt: 1 }} />
              <Giscus
                id="comments"
                repo="cloudreve/cloudreve"
                repoId="MDEwOlJlcG9zaXRvcnkxMjAxNTYwNzY="
                mapping={"number"}
                term={i18next.language == "zh-CN" ? "2170" : "2169"}
                reactionsEnabled={"1"}
                emitMetadata={"0"}
                inputPosition={"bottom"}
                theme={theme.palette.mode === "dark" ? "dark" : "light"}
                lang={i18next.language == "zh-CN" ? "zh-CN" : "en"}
                loading={"lazy"}
              />
            </StyledPaper>
          </Grid>
        </Grid>
      </Container>
    </PageContainer>
  );
};

export default Home;
