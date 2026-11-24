import type { MemoVO, ResultVO, SysConfigVO } from "~/types"
import CryptoJS from "crypto-js"
import { toast } from "vue-sonner"
import { useGlobalState } from "~/store"
import markdownit from "markdown-it"

const global = useGlobalState()

export const useMyFetch = async <T>(url: string, data?: any) => {
  const headers: Record<string, string> = {}

  const userinfo = global.value.userinfo
  if (userinfo.token) {
    headers["x-api-token"] = userinfo.token
  }

  const res = await $fetch<ResultVO<T>>(`/api${url}`, {
    method: "post",
    body: data ? JSON.stringify(data) : null,
    headers: headers,
  })

  if (!res || res.code !== 0) {
    if (!res) {
      throw new Error("请求失败")
    }

    if (res.code === 3 || res.code === 4) {
      global.value.userinfo = {}
      window.location.href = "/"
      throw new Error(res.message || "请求失败")
    }

    throw new Error(res.message)
  }

  return res.data
}

export const sha256 = async (file: File) => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.readAsArrayBuffer(file)
    reader.onload = () => {
      if (!reader.result) {
        reject(new Error("File read failed"))
        return
      }

      if (typeof reader.result === "string") {
        resolve(CryptoJS.SHA256(reader.result).toString(CryptoJS.enc.Hex))
        return
      }

      const wordArray = CryptoJS.lib.WordArray.create(reader.result)
      resolve(CryptoJS.SHA256(wordArray).toString(CryptoJS.enc.Hex))
    }
  })
}

type OnProgressCallback = (progress: number) => void

type OnTotalProgressCallback = (
  totalCount: number,
  currentCount: number,
  name: string,
  progress: number,
) => void

const upload2S3WithProgress = async (
  preSignedUrl: string,
  file: File,
  onProgress: OnProgressCallback,
): Promise<void> =>
  new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    xhr.addEventListener("load", () => {
      resolve()
    })
    xhr.addEventListener("error", () => reject(new Error("File upload failed")))
    xhr.addEventListener("abort", () =>
      reject(new Error("File upload aborted")),
    )
    xhr.upload.addEventListener("progress", e => onProgress(e.loaded / e.total))

    xhr.open("PUT", preSignedUrl, true)
    xhr.send(file)
  })

const upload2S3 = async (
  files: FileList,
  onProgress?: OnTotalProgressCallback,
): Promise<string[]> => {
  const result: string[] = []

  for (let i = 0; i < files.length; i++) {
    try {
      const res = await useMyFetch<{
        preSignedUrl: string
        imageUrl: string
      }>("/file/s3PreSigned", {
        contentType: files[0].type,
      })

      if (!res || !res.preSignedUrl) {
        toast.error("获取 S3 上传地址失败")
        continue
      }

      const file = files[i]
      await upload2S3WithProgress(res.preSignedUrl, file, progress => {
        if (onProgress) {
          onProgress(files.length, i + 1, file.name, progress)
        }
      })
      result.push(res.imageUrl)
    } catch (err) {
      toast.error(`上传文件到 S3 失败, ${err}`)
    }
  }

  return result
}

const uploadFile2ServerWithProgress = (
  url: string,
  file: File,
  onProgress: OnProgressCallback,
): Promise<string[]> =>
  new Promise<string[]>((resolve, reject) => {
    const xhr = new XMLHttpRequest()

    xhr.addEventListener("load", () => {
      const res = JSON.parse(xhr.responseText)
      if (!res || res.code !== 0) {
        return reject(new Error(`${res?.message || "请求失败"}`))
      }

      resolve(res.data || [])
    })
    xhr.addEventListener("error", () => reject(new Error("File upload failed")))
    xhr.addEventListener("abort", () =>
      reject(new Error("File upload aborted")),
    )
    xhr.upload.addEventListener("progress", e => onProgress(e.loaded / e.total))

    xhr.open("POST", url, true)

    const userinfo = global.value.userinfo
    if (userinfo.token) {
      xhr.setRequestHeader("x-api-token", userinfo.token)
    }

    const formData = new FormData()
    formData.append("files", file)

    xhr.send(formData)
  })

const uploadFile2Server = async (
  files: FileList,
  onProgress?: OnTotalProgressCallback,
): Promise<string[]> => {
  const result: string[] = []

  for (let i = 0; i < files.length; i++) {
    try {
      const file = files[i]
      if (onProgress) {
        onProgress(files.length, i + 1, file.name, 0)
      }

      // const hash = await sha256(file)
      // const ext = file.name.split(".").pop()
      // const filename = `${hash}.${ext}`
      // const res = await useMyFetch<{ exist: boolean, path: string }>(`/file/exist?filename=${filename}`)
      // if (res.exist) {
      //   result.push(res.path)
      //   if (onProgress) {
      //     onProgress(files.length, i + 1, file.name, 1)
      //   }

      //   continue
      // }

      const urlList = await uploadFile2ServerWithProgress(
        "/api/file/upload",
        file,
        progress => {
          if (onProgress) {
            onProgress(files.length, i + 1, file.name, progress)
          }
        },
      )

      if (!urlList.length) {
        toast.error(`上传文件到服务器失败`)
        continue
      }

      result.push(...urlList)
    } catch (e) {
      toast.error(`上传文件到服务器失败, ${e}`)
    }
  }

  return result
}

export const useUpload = async (
  files: FileList,
  onProgress?: OnTotalProgressCallback,
): Promise<string[]> => {
  if (files.length === 0) {
    toast.error("没有选择文件")
    return []
  }

  const sysConfig = useState<SysConfigVO>("sysConfig")
  if (sysConfig.value.enableS3) {
    return upload2S3(files, onProgress)
  }

  return uploadFile2Server(files, onProgress)
}

export const md = markdownit({
  html: true,
  linkify: true,
  typographer: true,
  breaks: true,
})

export const shikiReady = ref(0)
let shikiHighlighterPromise: Promise<any> | null = null
let shikiPluginInstalled = false
const loadedShikiLangs = new Set<string>()

const SHIKI_LANG_LOADERS: Record<string, () => Promise<any>> = {
  c: () => import("shiki/langs/c.mjs"),
  css: () => import("shiki/langs/css.mjs"),
  html: () => import("shiki/langs/html.mjs"),
  javascript: () => import("shiki/langs/javascript.mjs"),
  js: () => import("shiki/langs/javascript.mjs"),
  json: () => import("shiki/langs/json.mjs"),
  python: () => import("shiki/langs/python.mjs"),
  py: () => import("shiki/langs/python.mjs"),
  shellscript: () => import("shiki/langs/shellscript.mjs"),
  sh: () => import("shiki/langs/shellscript.mjs"),
  bash: () => import("shiki/langs/shellscript.mjs"),
  shell: () => import("shiki/langs/shellscript.mjs"),
  zsh: () => import("shiki/langs/shellscript.mjs"),
  sql: () => import("shiki/langs/sql.mjs"),
  tsx: () => import("shiki/langs/tsx.mjs"),
  ts: () => import("shiki/langs/tsx.mjs"),
  typescript: () => import("shiki/langs/tsx.mjs"),
  xml: () => import("shiki/langs/xml.mjs"),
  yaml: () => import("shiki/langs/yaml.mjs"),
  yml: () => import("shiki/langs/yaml.mjs"),
  go: () => import("shiki/langs/go.mjs"),
  golang: () => import("shiki/langs/go.mjs"),
}

export const extractCodeFenceLangs = (text?: string): string[] => {
  if (!text) return []
  const langs = new Set<string>()
  const re = /(?:^|\n)\s*(?:```|~~~)\s*([a-zA-Z0-9_-]*)/g
  let m: RegExpExecArray | null
  while ((m = re.exec(text)) !== null) {
    const lang = (m[1] || "").toLowerCase()
    if (lang && SHIKI_LANG_LOADERS[lang]) {
      langs.add(lang)
    }
  }
  return Array.from(langs)
}

export const hasCodeFence = (text?: string) =>
  Boolean(text && /(^|\n)\s*(```|~~~)/.test(text))

export const ensureShiki = async (langs: string[] = []): Promise<void> => {
  try {
    if (!shikiHighlighterPromise) {
      shikiHighlighterPromise = (async () => {
        const [{ createHighlighterCore }, { createJavaScriptRegexEngine }] =
          await Promise.all([
            import("shiki/core"),
            import("shiki/engine/javascript"),
          ])
        return createHighlighterCore({
          themes: [import("shiki/themes/github-dark.mjs")],
          langs: [],
          engine: createJavaScriptRegexEngine(),
        })
      })().catch(err => {
        shikiHighlighterPromise = null
        throw err
      })
    }

    const highlighter = await shikiHighlighterPromise
    let changed = false

    const missingLangs = langs.filter(
      l => SHIKI_LANG_LOADERS[l] && !loadedShikiLangs.has(l),
    )
    if (missingLangs.length > 0) {
      await highlighter.loadLanguage(
        ...missingLangs.map(l => SHIKI_LANG_LOADERS[l]()),
      )
      missingLangs.forEach(l => loadedShikiLangs.add(l))
      changed = true
    }

    if (!shikiPluginInstalled) {
      const { fromHighlighter } = await import("@shikijs/markdown-it/core")
      md.use(
        //@ts-ignore
        fromHighlighter(highlighter, {
          themes: {
            light: "github-dark",
            dark: "github-dark",
          },
          fallbackLanguage: "text",
        }),
      )
      shikiPluginInstalled = true
      changed = true
    }

    if (changed) {
      shikiReady.value += 1
    }
  } catch (err) {
    console.error("Failed to initialize Shiki highlighter", err)
  }
}

let musicAssetsPromise: Promise<void> | null = null

const loadExternalScript = (src: string) =>
  new Promise<void>((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) {
      resolve()
      return
    }
    const script = document.createElement("script")
    script.src = src
    script.async = true
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`Failed to load ${src}`))
    document.head.appendChild(script)
  })

export const ensureMusicAssets = (): Promise<void> => {
  if (!import.meta.client) return Promise.resolve()
  if (musicAssetsPromise) return musicAssetsPromise

  musicAssetsPromise = (async () => {
    if (!document.querySelector('link[href="/css/APlayer.min.css"]')) {
      const link = document.createElement("link")
      link.rel = "stylesheet"
      link.href = "/css/APlayer.min.css"
      document.head.appendChild(link)
    }
    await loadExternalScript("/js/APlayer.min.js")
    await loadExternalScript("/js/Meting.min.js")
  })().catch(err => {
    musicAssetsPromise = null
    console.error("Failed to load music player assets", err)
  })

  return musicAssetsPromise
}

export type MemoListResult = {
  list: Array<MemoVO>
  total: number
  hasNext: boolean
}

let prefetchedHomeMemos: Promise<MemoListResult> | null = null

export const prefetchHomeMemos = () => {
  if (!prefetchedHomeMemos) {
    prefetchedHomeMemos = useMyFetch<MemoListResult>("/memo/list", {
      page: 1,
      size: 10,
    }).catch(err => {
      prefetchedHomeMemos = null
      throw err
    })
  }
  return prefetchedHomeMemos
}

export const consumePrefetchedHomeMemos = (): Promise<MemoListResult> | null => {
  const pending = prefetchedHomeMemos
  prefetchedHomeMemos = null
  return pending
}
