import { Typography } from "@mui/material";
import dayjs from "dayjs";
import duration from "dayjs/plugin/duration";
import { TFunction } from "i18next";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse, Metadata } from "../../../api/explorer.ts";
import { formatLocalTime } from "../../../util/datetime.ts";
import CameraFilled from "../../Icons/CameraFilled.tsx";
import CameraRounded from "../../Icons/CameraRounded.tsx";
import ClockFilled from "../../Icons/ClockFilled.tsx";
import Copyright from "../../Icons/Copyright.tsx";
import DarkTheme from "../../Icons/DarkTheme.tsx";
import Image from "../../Icons/Image.tsx";
import InfoFilled from "../../Icons/InfoFilled.tsx";
import LocationFilled from "../../Icons/LocationFilled.tsx";
import MusicNote1 from "../../Icons/MusicNote1.tsx";
import Notepad from "../../Icons/Notepad.tsx";
import Person from "../../Icons/Person.tsx";
import WindowApps from "../../Icons/WindowApps.tsx";
import MapLoader from "./Map/MapLoader.tsx";
import MediaMetaCard, { MediaMetaContent, MediaMetaElements } from "./MediaMetaCard.tsx";

dayjs.extend(duration);

export interface MediaInfoProps {
  target: FileResponse;
}

const formatBitrate = (bits: string): string => {
  if (!bits) return "";

  const bitrate = parseFloat(bits);
  if (isNaN(bitrate)) return bits;

  if (bitrate < 1000) {
    return `${bitrate.toFixed(0)} bps`;
  } else if (bitrate < 1000000) {
    return `${(bitrate / 1000).toFixed(1)} kbps`;
  } else {
    return `${(bitrate / 1000000).toFixed(2)} mbps`;
  }
};

export const getAperture = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.f_number]) {
    const fInt = parseFloat(target.metadata[Metadata.f_number]);
    if (fInt) {
      return {
        display: `ƒ/${fInt.toFixed(1)}`,
        searchKey: Metadata.f_number,
        searchValue: target.metadata[Metadata.f_number],
      };
    }
  }
};

export const getExposure = (target: FileResponse, t: TFunction): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.exposure_time]) {
    return {
      display: t("application:fileManager.exposureValue", {
        num: target.metadata[Metadata.exposure_time],
      }),
      searchKey: Metadata.exposure_time,
      searchValue: target.metadata[Metadata.exposure_time],
    };
  }
};

export const getIso = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.iso_speed_ratings]) {
    return {
      display: target.metadata[Metadata.iso_speed_ratings],
      searchKey: Metadata.iso_speed_ratings,
      searchValue: target.metadata[Metadata.iso_speed_ratings],
    };
  }
};

export const getCameraMake = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.camera_make]) {
    return {
      display: target.metadata[Metadata.camera_make],
      searchKey: Metadata.camera_make,
      searchValue: target.metadata[Metadata.camera_make],
    };
  }
};

export const getCameraModel = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.camera_model]) {
    return {
      display: target.metadata[Metadata.camera_model],
      searchKey: Metadata.camera_model,
      searchValue: target.metadata[Metadata.camera_model],
    };
  }
};

export const getLensMake = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.lens_make]) {
    return {
      display: target.metadata[Metadata.lens_make],
      searchKey: Metadata.lens_make,
      searchValue: target.metadata[Metadata.lens_make],
    };
  }
};

export const getLensModel = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.lens_model]) {
    return {
      display: target.metadata[Metadata.lens_model],
      searchKey: Metadata.lens_model,
      searchValue: target.metadata[Metadata.lens_model],
    };
  }
};

export const getFocalLength = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.focal_length]) {
    return {
      display: `${target.metadata[Metadata.focal_length]}mm`,
      searchKey: Metadata.focal_length,
      searchValue: target.metadata?.[Metadata.focal_length],
    };
  }
};

export const getExposureBias = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.exposure_bias_value]) {
    const evFloat = parseFloat(target.metadata[Metadata.exposure_bias_value]);
    return {
      display: `${evFloat.toFixed(1)} ev`,
      searchKey: Metadata.exposure_bias_value,
      searchValue: target.metadata[Metadata.exposure_bias_value],
    };
  }
};

export const getFlash = (target: FileResponse, t: TFunction): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.flash]) {
    return {
      display: target.metadata[Metadata.flash] == "1" ? t("fileManager.on") : t("fileManager.off"),
      searchKey: Metadata.flash,
      searchValue: target.metadata[Metadata.flash],
    };
  }
};

export const getSoftware = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.software]) {
    return {
      display: target.metadata[Metadata.software],
      searchKey: Metadata.software,
      searchValue: target.metadata[Metadata.software],
    };
  }
};

export const takenAt = (target: FileResponse): string | undefined => {
  if (target.metadata?.[Metadata.taken_at]) {
    return formatLocalTime(target.metadata[Metadata.taken_at], true);
  }
};

export const getImageSize = (target: FileResponse): (MediaMetaElements | string)[] | undefined => {
  if (!target.metadata?.[Metadata.pixel_x_dimension] || !target.metadata?.[Metadata.pixel_y_dimension]) {
    return undefined;
  }

  const holder: (MediaMetaElements | string)[] = [];
  const x = parseInt(target.metadata[Metadata.pixel_x_dimension]);
  const y = parseInt(target.metadata[Metadata.pixel_y_dimension]);
  const mp = (x * y) / 1000000;
  if (mp > 0.1) {
    holder.push(`${mp.toFixed(1)} MP · `);
  }
  holder.push({
    display: `${x}`,
    searchKey: Metadata.pixel_x_dimension,
    searchValue: target.metadata[Metadata.pixel_x_dimension],
  });
  holder.push(" x ");
  holder.push({
    display: `${y}`,
    searchKey: Metadata.pixel_y_dimension,
    searchValue: target.metadata[Metadata.pixel_y_dimension],
  });

  return holder;
};

export const getMediaTitle = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.music_title]) {
    return {
      display: target.metadata[Metadata.music_title],
      searchKey: Metadata.music_title,
      searchValue: target.metadata[Metadata.music_title],
    };
  } else if (target.metadata?.[Metadata.stream_title]) {
    return {
      display: target.metadata[Metadata.stream_title],
      searchKey: Metadata.stream_title,
      searchValue: target.metadata[Metadata.stream_title],
    };
  }
};

export const getArtist = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.music_artist]) {
    return {
      display: target.metadata[Metadata.music_artist],
      searchKey: Metadata.music_artist,
      searchValue: target.metadata[Metadata.music_artist],
    };
  }
};

export const getAlbum = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.music_album]) {
    return {
      display: target.metadata[Metadata.music_album],
      searchKey: Metadata.music_album,
      searchValue: target.metadata[Metadata.music_album],
    };
  }
};

export const getDuration = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.stream_duration]) {
    return {
      display: dayjs.duration(parseFloat(target.metadata[Metadata.stream_duration]), "seconds").format("HH:mm:ss"),
      searchKey: Metadata.stream_duration,
      searchValue: target.metadata[Metadata.stream_duration],
    };
  }
};

export const getStreet = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.street]) {
    return {
      display: target.metadata[Metadata.street],
      searchKey: Metadata.street,
      searchValue: target.metadata[Metadata.street],
    };
  }
};

export const getLocality = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.locality]) {
    return {
      display: target.metadata[Metadata.locality],
      searchKey: Metadata.locality,
      searchValue: target.metadata[Metadata.locality],
    };
  }
};

export const getPlace = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.place]) {
    return {
      display: target.metadata[Metadata.place],
      searchKey: Metadata.place,
      searchValue: target.metadata[Metadata.place],
    };
  }
};

export const getDistrict = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.district]) {
    return {
      display: target.metadata[Metadata.district],
      searchKey: Metadata.district,
      searchValue: target.metadata[Metadata.district],
    };
  }
};

export const getRegion = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.region]) {
    return {
      display: target.metadata[Metadata.region],
      searchKey: Metadata.region,
      searchValue: target.metadata[Metadata.region],
    };
  }
};

export const getCountry = (target: FileResponse): MediaMetaElements | undefined => {
  if (target.metadata?.[Metadata.country]) {
    return {
      display: target.metadata[Metadata.country],
      searchKey: Metadata.country,
      searchValue: target.metadata[Metadata.country],
    };
  }
};

const MediaInfo = ({ target }: MediaInfoProps) => {
  if (!target.metadata) {
    return undefined;
  }

  const { t } = useTranslation();

  const exifContents = useMemo(() => {
    let res: MediaMetaContent[] = [];
    const aperture = getAperture(target);
    if (aperture) {
      res.push({
        title: [t("fileManager.aperture")],
        content: [aperture],
      });
    }

    const exposure = getExposure(target, t);
    if (exposure) {
      res.push({
        title: [t("fileManager.exposure")],
        content: [exposure],
      });
    }

    const iso = getIso(target);
    if (iso) {
      res.push({
        title: [t("fileManager.iso")],
        content: [iso],
      });
    }

    return res;
  }, [target, t]);

  const cameraModelContent = useMemo(() => {
    let title: (MediaMetaElements | string)[] = [];
    let content: (MediaMetaElements | string)[] = [];

    const cameraMake = getCameraMake(target);
    if (cameraMake) {
      title.push(cameraMake);
    }

    const cameraModel = getCameraModel(target);
    if (cameraModel) {
      if (title.length > 0) {
        title.push(" ");
      }
      title.push(cameraModel);
    }

    const lensMake = getLensMake(target);
    if (lensMake) {
      content.push(lensMake);
    }

    const lensModel = getLensModel(target);
    if (lensModel) {
      if (content.length > 0) {
        content.push(" ");
      }
      content.push(lensModel);
    }

    const focalLength = getFocalLength(target);
    if (focalLength) {
      // Push ( to the start of the content
      const contentEmpty = content.length == 0;
      if (!contentEmpty) {
        content.unshift(" (");
      }
      content.unshift(focalLength);
      if (!contentEmpty) {
        content.push(")");
      }
    }
    if (title.length == 0) {
      title = content;
    }
    if (title.length > 0 || content.length > 0) {
      return { title, content };
    }
    return undefined;
  }, [target, t]);

  const lightInfoContent = useMemo(() => {
    let res: MediaMetaContent[] = [];

    const exposureBias = getExposureBias(target);
    if (exposureBias) {
      res.push({
        title: [t("fileManager.exposureBias")],
        content: [exposureBias],
      });
    }

    const flash = getFlash(target, t);
    if (flash) {
      res.push({
        title: [t("fileManager.flash")],
        content: [flash],
      });
    }
    return res;
  }, [target, t]);

  const copyRightContent = useMemo(() => {
    const holder: (MediaMetaElements | string)[] = [];
    if (target.metadata?.[Metadata.copy_right]) {
      holder.push({
        display: target.metadata[Metadata.copy_right],
        searchKey: Metadata.copy_right,
        searchValue: target.metadata[Metadata.copy_right],
      });
    }

    if (target.metadata?.[Metadata.artist]) {
      if (holder.length > 0) {
        holder.push(" ");
      }
      holder.push({
        display: target.metadata[Metadata.artist],
        searchKey: Metadata.artist,
        searchValue: target.metadata[Metadata.artist],
      });
    }

    if (holder.length == 0) {
      return undefined;
    }

    return {
      title: [t("fileManager.copyright")],
      content: holder,
    } as MediaMetaContent;
  }, [target, t]);

  const softwareContent = useMemo(() => {
    const software = getSoftware(target);
    if (!software) {
      return undefined;
    }

    return {
      title: [t("fileManager.software")],
      content: [software],
    } as MediaMetaContent;
  }, [target, t]);

  const mapBoxGps = useMemo(() => {
    if (!target.metadata?.[Metadata.gps_lat] || !target.metadata?.[Metadata.gps_lng]) {
      return undefined;
    }

    return [parseFloat(target.metadata[Metadata.gps_lat]), parseFloat(target.metadata[Metadata.gps_lng])] as [
      number,
      number,
    ];
  }, [target, t]);

  const takenTimeContent = useMemo(() => {
    const takenTime = takenAt(target);
    if (!takenTime) {
      return undefined;
    }

    return {
      title: [t("application:fileManager.takenAt")],
      content: [takenTime],
    } as MediaMetaContent;
  }, [target, t]);

  const imageSizeContent = useMemo(() => {
    const imageSize = getImageSize(target);
    if (!imageSize) {
      return undefined;
    }

    return {
      title: [t("fileManager.resolution")],
      content: imageSize,
    } as MediaMetaContent;
  }, [target, t]);

  const mediaTitleContent = useMemo(() => {
    const res = {
      title: [t("fileManager.title")],
      content: [],
    } as MediaMetaContent;

    const mediaTitle = getMediaTitle(target);
    if (mediaTitle) {
      res.content.push(mediaTitle);
      return res;
    } else {
      return undefined;
    }
  }, [target, t]);

  const musicArtistContent = useMemo(() => {
    const artist = getArtist(target);
    if (!artist) {
      return undefined;
    }
    return {
      title: [t("fileManager.artist")],
      content: [artist],
    } as MediaMetaContent;
  }, [target, t]);

  const albumContent = useMemo(() => {
    const album = getAlbum(target);
    if (!album) {
      return undefined;
    }

    return {
      title: [t("fileManager.album")],
      content: [album],
    } as MediaMetaContent;
  }, [target, t]);

  const durationContent = useMemo(() => {
    const duration = getDuration(target);
    if (!duration) {
      return undefined;
    }

    return {
      title: [t("fileManager.duration")],
      content: [duration],
    } as MediaMetaContent;
  }, [target, t]);

  const streamFormatContent = useMemo(() => {
    let res: MediaMetaContent[] = [];

    if (target.metadata?.[Metadata.stream_format_long]) {
      res.push({
        title: [t("fileManager.format")],
        content: [
          {
            display: target.metadata[Metadata.stream_format_long],
            searchKey: Metadata.stream_format_long,
            searchValue: target.metadata[Metadata.stream_format_long],
          },
        ],
      });

      if (target.metadata?.[Metadata.stream_bit_rate]) {
        res[0].content.push(" · ");
        res[0].content.push({
          display: formatBitrate(target.metadata[Metadata.stream_bit_rate]),
          searchKey: Metadata.stream_bit_rate,
          searchValue: target.metadata[Metadata.stream_bit_rate],
        });
      }
    }

    if (res.length == 0) {
      return undefined;
    }

    return res;
  }, [target, t]);

  const singleStreamContents = useMemo(() => {
    let res: MediaMetaContent[] = [];
    const allMetas = Object.keys(target.metadata ?? {});
    const streamGrouped: {
      [key: string]: { type: string; [key: string]: string };
    } = {};
    allMetas.forEach((meta) => {
      if (!meta.startsWith("stream:stream_")) {
        return;
      }

      const [prefix, group, type, ...other] = meta.split("_");
      if (!streamGrouped[group]) {
        streamGrouped[group] = {
          type: type,
        };
      }

      if (type != "video" && type != "audio") {
        return;
      }

      streamGrouped[group][other.join("_")] = target.metadata?.[meta] ?? "";
    });

    for (const [group, item] of Object.entries(streamGrouped)) {
      let content: string[] = [];
      if (item[Metadata.stream_indexed_codec]) {
        content.push(item[Metadata.stream_indexed_codec]);
      }
      if (item[Metadata.stream_indexed_bitrate]) {
        content.push(formatBitrate(item[Metadata.stream_indexed_bitrate]));
      }
      if (item[Metadata.stream_indexed_width] && item[Metadata.stream_indexed_height]) {
        content.push(`${item[Metadata.stream_indexed_width]}x${item[Metadata.stream_indexed_height]}`);
      }
      if (content.length == 0) {
        continue;
      }
      res.push({
        title: [`Stream #${group} (${item.type})`],
        content: [content.join(" · ")],
      });
    }

    if (res.length > 0) {
      return res;
    }

    return undefined;
  }, [target, t]);

  const gpsAddressContent = useMemo(() => {
    let content: (MediaMetaElements | string)[] = [];

    // Build address components in hierarchical order (most specific to least specific)
    const addressComponents: (MediaMetaElements | undefined)[] = [];

    const street = getStreet(target);
    if (street) {
      addressComponents.push(street);
    }
    const locality = getLocality(target);
    if (locality) {
      addressComponents.push(locality);
    }
    const place = getPlace(target);
    if (place) {
      addressComponents.push(place);
    }
    const district = getDistrict(target);
    if (district) {
      addressComponents.push(district);
    }
    const region = getRegion(target);
    if (region) {
      addressComponents.push(region);
    }
    const country = getCountry(target);
    if (country) {
      addressComponents.push(country);
    }

    // Filter out undefined components and join with commas
    const validComponents = addressComponents.filter(Boolean) as MediaMetaElements[];

    if (validComponents.length > 0) {
      // Add the first component
      content.push(validComponents[0]);

      // Add remaining components with comma separators
      for (let i = 1; i < validComponents.length; i++) {
        content.push(", ");
        content.push(validComponents[i]);
      }

      return { title: [t("fileManager.address")], content };
    }

    return undefined;
  }, [target, t]);

  const showExifBasic = exifContents.length > 0;
  const showLightInfo = lightInfoContent.length > 0;
  const showCopyRight = !!copyRightContent;
  const showCameraModel = !!cameraModelContent;
  const showMediaInfo =
    showExifBasic ||
    showCameraModel ||
    showLightInfo ||
    showCopyRight ||
    softwareContent ||
    takenTimeContent ||
    imageSizeContent ||
    mediaTitleContent ||
    musicArtistContent ||
    durationContent ||
    streamFormatContent ||
    singleStreamContents ||
    mapBoxGps ||
    gpsAddressContent;

  if (!showMediaInfo) {
    return undefined;
  }

  return (
    <>
      <Typography sx={{ pt: 1 }} color="textPrimary" fontWeight={500} variant={"subtitle1"}>
        {t("fileManager.mediaInfo")}
      </Typography>
      {showExifBasic && <MediaMetaCard icon={CameraRounded} contents={exifContents} />}
      {showLightInfo && <MediaMetaCard icon={DarkTheme} contents={lightInfoContent} />}
      {showCameraModel && <MediaMetaCard icon={CameraFilled} contents={[cameraModelContent]} />}
      {takenTimeContent && <MediaMetaCard icon={ClockFilled} contents={[takenTimeContent]} />}
      {imageSizeContent && <MediaMetaCard icon={Image} contents={[imageSizeContent]} />}
      {showCopyRight && <MediaMetaCard icon={Copyright} contents={[copyRightContent]} />}
      {softwareContent && <MediaMetaCard icon={WindowApps} contents={[softwareContent]} />}
      {mapBoxGps && <MapLoader height={200} center={mapBoxGps} />}
      {gpsAddressContent && <MediaMetaCard icon={LocationFilled} contents={[gpsAddressContent]} />}
      {mediaTitleContent && <MediaMetaCard icon={Notepad} contents={[mediaTitleContent]} />}
      {musicArtistContent && <MediaMetaCard icon={Person} contents={[musicArtistContent]} />}
      {albumContent && <MediaMetaCard icon={MusicNote1} contents={[albumContent]} />}
      {durationContent && <MediaMetaCard icon={ClockFilled} contents={[durationContent]} />}
      {streamFormatContent && <MediaMetaCard icon={InfoFilled} contents={streamFormatContent} />}
      {singleStreamContents && singleStreamContents.map((content) => <MediaMetaCard contents={[content]} />)}
    </>
  );
};

export default MediaInfo;
