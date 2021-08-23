module.exports = {
  title: 'Ostracon',
  base: process.env.VUEPRESS_BASE,
  head: [
    ['script', { src: 'https://polyfill.io/v3/polyfill.min.js?features=es6' }],
    ['script', { id: 'MathJax-script', src: 'https://cdn.jsdelivr.net/npm/mathjax@3/es5/tex-mml-chtml.js', async: "async"}],
    ['script', { }, 'window.MathJax = { tex: { inlineMath: [[\'$\',\'$\'], [\'\\\\(\',\'\\\\)\']] } };']
  ],
  themeConfig: {
    repo: 'line/ostracon',
    docsRepo: 'line/ostracon',
    docsDir: 'docs',
    label: 'core',
    versions: [
      {
        "label": "main",
        "key": "main"
      }
    ],
    search: false,
    sidebar: 'auto',
    footer: {
      textLink: {
        text: 'LINE Blockchain(blockchain.line.me)',
        url: 'https://blockchain.line.me/'
      },
      services: [
        {
          service: 'medium',
          url: 'https://lineblockchain.medium.com'
        },
        {
          service: 'note',
          url: 'https://note.com/line_blockchain'
        },
        {
          service: 'twitter',
          url: 'https://twitter.com/LINEBC_Global'
        },
        {
          service: 'twitter JP',
          url: 'https://twitter.com/linebc_japan'
        }
      ],
      links: [
        {
          title: 'Contributing',
          children: [
            {
              title: 'Source code on GitHub',
              url: 'https://github.com/line/ostracon'
            }
          ]
        }
      ]
    }
  }
};
