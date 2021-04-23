<?php
/* Astronomy Picture of the Day server app for
   Ataris with #FujiNet devices.

   Have the Atari read (e.g., "OPEN #1,4,0,..." in BASIC)
   from this script (N:HTTP://server/path/to/index.php)
   with the following GET arguments:

    * ?mode=9 -- fetch 80x192 GRAPHICS 9 16 greyscale image
    * ?mode=15 -- fetch 160x192 GRAPHICS 15 4 color image *
    * ?mode=8 -- fetch 320x192 GRAPHICS 8 black & white image

   Read the 7,680 bytes (40 x 192, aka 30 pages) into screen
   memory.  You can then read until an end-of-line or the
   end-of-file to grab the title and description of the image
   (e.g., "INPUT #1,A$").  (Mode 15 will return four additional
   bytes of color palette data, for COLOR4 (background), and
   COLOR0, COLOR1, and COLOR2 (foreground).)

   Other more complicated modes:

    * ?mode=rgb9 -- fetch 80x192 GRAPHICS 9 4096 color (R, G, B split)

   Sample options:

    * ?sample=N -- fetch a sample image, rather than APOD (where N is 1 or higher)

   Date options:

    * ?date=YYMMDD -- fetch the APOD for a given day
      (if provided, will fetch from https://apod.nasa.gov/apod/apYYMMDD.html;
      if not provided, will fetch from https://apod.nasa.gov/apod/astropix.html)
*/

$sample_files = array(
  "alt_reality.png",
  "ngc2818.jpg",
  "Parrot.jpg",
  "SPACE.JPG",
  "rainbow.png"
);

$date = new DateTime("now", new DateTimeZone('America/New_York') );

$date_wanted = NULL;

if (array_key_exists("date", $_GET)) {
  $date_request = $_GET["date"];
  if (preg_match("/^([0-9][0-9])([0-9][0-9])([0-9][0-9])$/", $date_request, $matches)) {
    $yr = $matches[1];
    $mo = $matches[2];
    $da = $matches[3];

    $date_wanted = sprintf("%02d%02d%02d", $yr, $mo, $da);
  }
} else {
  $today = $date->format("Y-m-d");
}

if (array_key_exists("sample", $_GET) &&
    intval($_GET["sample"]) &&
    intval($_GET["sample"]) <= count($sample_files)
) {
  $sample = intval($_GET["sample"]);
  $basename = "SAMPLE" . $sample;
} else {
  $sample = false;

  if ($date_wanted !== NULL) {
    $basename = "AP" . $date_wanted;
  } else {
    $basename = "AP" . date("ymd");
  }
}

/* What mode of image do they want? */
if (array_key_exists("mode", $_GET)) {
  $mode = trim($_GET["mode"]);
} else {
  $mode = "";
}

if ($mode == "8") {
  $img_size = 7680;
  $pal_size = 0;
  $outfile = "img/$basename.GR8";
} else if ($mode == "15") {
  $img_size = 7680;
  $pal_size = 4;
  $outfile = "img/$basename.G15";
} else if ($mode == "rgb9") {
  $img_size = 7680 * 3;
  $pal_size = 0;
  $outfile = "img/$basename.CV9";
} else {
  $img_size = 7680;
  $pal_size = 0; /* FIXME: Would be nice to pick a suitable hue and send it down the wire */
  $mode = "9";
  $outfile = "img/$basename.GR9";
}


if (!$sample) {
  /* Check whether it's a new day, and we'll need
     to fetch and convert an the image */
  if (!file_exists($outfile)) {
    /* Time to fetch a new one */
    $img_src = "";
    if ($date_wanted !== NULL) {
      $url = "https://apod.nasa.gov/apod/ap" . $date_wanted . ".html";
    } else {
      $url = "https://apod.nasa.gov/apod/astropix.html";
    }
    $page = file_get_contents($url);

    if (!empty($page)) {
      $dom = new DOMDocument;
      if ($dom->loadHTML($page)) {
        $imgs = $dom->getElementsByTagName('img');
        foreach ($imgs as $img) {
          if ($img_src == "") {
            $img_src = $img->getAttribute('src');
          }
        }
      }

      if ($img_src != "") {
        /* Found an image! Convert it! */
        system("./fetch_and_cvt.sh 'https://apod.nasa.gov/apod/$img_src' '$mode' '$outfile'");
      } else {
        /* Let's see if there's a YouTube video */
        $vid_src = "";

        $iframes = $dom->getElementsByTagName('iframe');
        foreach ($iframes as $iframe) {
          if ($vid_src == "") {
            if (stripos($iframe->getAttribute('src'), "https://www.youtube.com/embed/") !== false) {
              $vid_src = $iframe->getAttribute('src');
            }
          }
        }

        if ($vid_src != "") {
          /* Found a video! Fetch and convert its thumbnail! */
          $vid_url_path = parse_url($vid_src, PHP_URL_PATH);
          $vid_parts = explode("/", $vid_url_path);
          $vid_id = $vid_parts[2];

          if ($vid_id) {
            system("./fetch_and_cvt.sh 'https://img.youtube.com/vi/$vid_id/hqdefault.jpg' '$mode' '$outfile'");
          }
        }
      }
    }


    $rss = file_get_contents("https://apod.nasa.gov/apod.rss");
    if (!empty($rss)) {
      $dom = new DOMDocument;
      if ($dom->loadXML($rss)) {
        $items = $dom->getElementsByTagName('item');
        if ($items) {
          $latest = $items->item(0);

          if ($latest->childNodes) {
            foreach ($latest->childNodes as $child) {
              if ($child->tagName == "title") {
                $title = trim(preg_replace("/\s+/", " ", strip_tags($child->textContent)));
              }
              if ($child->tagName == "description") {
                $descr = trim(preg_replace("/\s+/", " ", strip_tags($child->textContent)));
              }
            }
          }
        }
      }

      $fo = fopen("descr.txt", "w");

      /* Store it, word-wrapping the title to avoid words
         breaking at the end of a line, but then pad each
         line to 40 characters, so we only need to INPUT
         one string (max 159 characters on the Atari end,
         to avoid scrolling any text off the 4-line text window) */
      $title = wordwrap($title, 40);
      $title_lines = explode("\n", $title);

      $title = "";
      foreach ($title_lines as $t) {
        $title .= $t;
        $pad = strlen($t) % 40;
        if ($pad != 0) {
          $title = $title . str_repeat("_", 40 - $pad);
        }
      }

      fprintf($fo, "%s", $title);
      fprintf($fo, "%s%c", $descr, 155); /* 155 = Atari EOL character */
      fclose($fo);
    }
  }
} else {
  /* This is kinda dumb, but `wget` can't fetch via `file` scheme */
  $sample_img = "http://billsgames.com/fujinet/apod/samples/" . $sample_files[$sample - 1];
  system("./fetch_and_cvt.sh '$sample_img' '$mode' '$outfile'");
}

/* Get the image */
$img = file_get_contents($outfile);
$descr = file_get_contents("descr.txt");


/* Dump the results: */
header("Content-Type: application/octet-stream");
header("Content-Length: " . ($img_size + $pal_size + strlen($descr)));
header("Content-Disposition: attachment; filename=\"" . basename($outfile) . "\"");

echo $img;
echo $descr;
