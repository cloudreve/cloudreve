import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import { Box, List, ListItem, ListItemText, Skeleton, styled } from "@mui/material";
import { useTranslation } from "react-i18next";
import { FullTextSearchResult as FullTextSearchResultType } from "../../../api/explorer.ts";
import { SecondaryButton } from "../../Common/StyledComponents.tsx";
import SearchResultItem from "./SearchResultItem.tsx";

export interface FullTextSearchResultProps {
  results: FullTextSearchResultType[];
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
}

const HighlightedContent = styled("span")({
  "& em": {
    fontStyle: "normal",
    backgroundColor: "#ffc1079e",
    borderRadius: "4px",
    boxShadow: "0 0 0 2px #ffc1079e",
  },
});

const contextChars = 80;

function trimContent(content: string): string {
  const emIdx = content.indexOf("<em>");
  if (emIdx === -1) {
    return content.length > contextChars * 2 ? content.slice(0, contextChars * 2) + "..." : content;
  }

  const start = Math.max(0, emIdx - contextChars);
  const emCloseIdx = content.indexOf("</em>", emIdx);
  const end = Math.min(content.length, (emCloseIdx !== -1 ? emCloseIdx + 5 : emIdx + 4) + contextChars);

  let trimmed = content.slice(start, end);

  // Remove partial <em> or </em> tags at the boundaries
  trimmed = trimmed.replace(/^(?:\/)?e?m?>/, "");
  trimmed = trimmed.replace(/<\/?e?m?$/, "");

  if (start > 0) trimmed = "..." + trimmed;
  if (end < content.length) trimmed = trimmed + "...";
  return trimmed;
}

function parseHighlightedContent(text: string): { text: string; highlighted: boolean }[] {
  // Merge adjacent <em> tags: <em>foo</em><em>bar</em> → <em>foobar</em>
  text = text.replace(/<\/em><em>/g, "");
  const segments: { text: string; highlighted: boolean }[] = [];
  const regex = /<em>([\s\S]*?)<\/em>/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      // Strip any orphaned <em> or </em> from plain text segments
      const plain = text.slice(lastIndex, match.index).replace(/<\/?em>/g, "");
      if (plain) segments.push({ text: plain, highlighted: false });
    }
    segments.push({ text: match[1], highlighted: true });
    lastIndex = regex.lastIndex;
  }

  if (lastIndex < text.length) {
    const plain = text.slice(lastIndex).replace(/<\/?em>/g, "");
    if (plain) segments.push({ text: plain, highlighted: false });
  }

  return segments;
}

const FullTextSearchResult = ({ results, loading, hasMore, onLoadMore }: FullTextSearchResultProps) => {
  const { t } = useTranslation();

  return (
    <List sx={{ width: "100%", px: 1 }} dense>
      {results.map((result) => (
        <SearchResultItem
          key={result.file.id}
          file={result.file}
          primary={result.file.name}
          secondary={
            <HighlightedContent>
              {parseHighlightedContent(trimContent(result.content)).map((segment, i) =>
                segment.highlighted ? <em key={i}>{segment.text}</em> : segment.text,
              )}
            </HighlightedContent>
          }
        />
      ))}
      {loading &&
        [0].map((i) => (
          <ListItem key={i} dense sx={{ px: 1 }}>
            <Box sx={{ display: "flex", alignItems: "center", width: "100%", gap: 1, py: 0.5 }}>
              <Skeleton variant="rounded" width={32} height={32} sx={{ flexShrink: 0 }} />
              <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                <Skeleton variant="text" width="40%" />
                <Skeleton variant="text" width="70%" />
              </Box>
            </Box>
          </ListItem>
        ))}
      {hasMore && !loading && (
        <ListItem disablePadding dense>
          <SecondaryButton fullWidth sx={{ py: 0.5, mt: 1, justifyContent: "center" }} onClick={onLoadMore}>
            <ExpandMoreIcon sx={{ fontSize: 18, mr: 0.5, color: "text.secondary" }} />
            <ListItemText
              primary={t("navbar.loadMore")}
              slotProps={{
                primary: {
                  variant: "body2",
                  color: "textSecondary",
                  textAlign: "center",
                },
              }}
            />
          </SecondaryButton>
        </ListItem>
      )}
    </List>
  );
};

export default FullTextSearchResult;
