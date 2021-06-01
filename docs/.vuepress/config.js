module.exports = {
  title: 'Ostracon',
  base: process.env.VUEPRESS_BASE,
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
