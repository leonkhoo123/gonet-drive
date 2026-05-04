import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "GoNet Drive",
  description: "A high-performance, self-hosted file management and media streaming server.",
  base: '/gonet-drive/',
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'GitHub', link: 'https://github.com/leonkhoo123/gonet-drive' }
    ],

    sidebar: [
      {
        text: 'Overview',
        items: [
          { text: 'Introduction', link: '/' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/leonkhoo123/gonet-drive' }
    ],
    
    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © Present - leonkhoo123'
    }
  }
})