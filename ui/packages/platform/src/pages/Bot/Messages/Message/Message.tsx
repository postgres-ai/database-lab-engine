import React, { useEffect, useMemo, useRef, useState } from 'react'
import ReactMarkdown, { Components } from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { makeStyles } from "@material-ui/core";
import { colors } from "@postgres.ai/shared/styles/colors";
import { icons } from "@postgres.ai/shared/styles/icons";
import { DebugDialog } from "../../DebugDialog/DebugDialog";
import { CodeBlock } from "./CodeBlock/CodeBlock";
import { disallowedHtmlTagsForMarkdown } from "../../utils";
import { MessageStatus, StateMessage } from "../../../../types/api/entities/bot";
import { MermaidDiagram } from "./MermaidDiagram/MermaidDiagram";
import { useAiBot } from "../../hooks";
import { ToolCallRenderer } from "./ToolCallRenderer/ToolCallRenderer";
import { transformAllCustomTags } from "../utils";
import { ThinkBlockRenderer } from './ThinkingCard/ThinkingCard';
import { MessageHeader } from "./MessageHeader/MessageHeader";


export type BaseMessageProps = {
  id: string | null;
  created_at?: string;
  content?: string;
  name?: string;
  isLoading?: boolean;
  formattedTime?: string;
  aiModel?: string;
  stateMessage?: StateMessage | null;
  isCurrentStreamMessage?: boolean;
  isPublic?: boolean;
  threadId?: string;
  status?: MessageStatus
}

type AiMessageProps = BaseMessageProps & {
  isAi: true;
  content: string;
  aiModel: string;
  isCurrentStreamMessage?: boolean;
}

type HumanMessageProps = BaseMessageProps & {
  isAi: false;
  name: string;
  content: string;
}

type LoadingMessageProps = BaseMessageProps & {
  isLoading: true;
  isAi: true;
  content?: undefined;
  stateMessage: StateMessage | null;
}

type MessageProps = AiMessageProps | HumanMessageProps | LoadingMessageProps;

const useStyles = makeStyles(
  (theme) => ({
    message: {
      padding: 10,
      paddingLeft: 60,
      position: 'relative',
      whiteSpace: 'normal',
      [theme.breakpoints.down('xs')]: {
        paddingLeft: 30
      },
      '& .markdown pre': {
        [theme.breakpoints.down('sm')]: {
          display: 'inline-block',
          minWidth: '100%',
          width: 'auto',
        },
        [theme.breakpoints.up('md')]: {
          display: 'block',
          maxWidth: 'auto',
          width: 'auto',
        },
        [theme.breakpoints.up('lg')]: {
          display: 'block',
          maxWidth: 'auto',
          width: 'auto',
        },
      },
    },
    messageAvatar: {
      top: '10px',
      left: '15px',
      position: 'absolute',
      width: 30,
      height: 30,
      [theme.breakpoints.down('xs')]: {
        width: 24,
        height: 24,
        left: 0,
        '& svg': {
          width: 24,
          height: 24,
        }
      }
    },
    messageAvatarImage: {
      width: '100%',
      borderRadius: '50%'
    },
    messageAuthor: {
      fontSize: 14,
      fontWeight: 'bold',
    },
    messageInfo: {
      display: 'inline-block',
      marginLeft: 10,
      padding: 0,
      fontSize: '0.75rem',
      color: colors.pgaiDarkGray,
      transition: '.2s ease',
      background: "none",
      border: "none",
      textDecoration: "none",
      '@media (max-width: 450px)': {
        '&:nth-child(1)': {
          display: 'none'
        }
      }
    },
    messageInfoActive: {
      borderBottom: '1px solid currentcolor',
      cursor: 'pointer',
      '&:hover': {
        color: '#404040'
      }
    },
    messageHeader: {
      height: '1.125rem',
      display: 'flex',
      flexWrap: 'wrap',
      alignItems: 'baseline',
      '@media (max-width: 450px)': {
        height: 'auto',
      }
    },
    additionalInfo: {
      '@media (max-width: 450px)': {
        width: '100%',
        marginTop: 4,
        marginLeft: -10,

      }
    },
    badge: {
      fontSize: '0.5rem',
      minWidth: '0.714rem',
      height: '0.714rem',
      padding: '0 0.25rem'
    },
    messagesSpinner: {
      display: 'flex',
      justifyContent: 'center',
      padding: 10
    },
    markdown: {
      margin: '5px 5px 5px 0',
      fontSize: 14,
      '& h1': {
        marginTop: 5
      },
      '& table': {
        borderCollapse: 'collapse',
        borderSpacing: 0
      },
      '& tr': {
        borderTop: '1px solid #c6cbd1',
        background: '#fff'
      },
      '& th, & td': {
        padding: '10px 13px',
        border: '1px solid #dfe2e5'
      },
      '& table tr:nth-child(2n)': {
        background: '#f6f8fa'
      },

      '& blockquote': {
        color: '#666',
        margin: 0,
        paddingLeft: '3em',
        borderLeft: '0.5em #eee solid'
      },
      '& img.emoji': {
        marginTop: 5
      },
      '& code': {
        border: '1px dotted silver',
        display: 'inline-block',
        borderRadius: 3,
        padding: 2,
        backgroundColor: '#f6f8fa',
        marginBottom: 3,
        fontSize: '13px !important',
        fontFamily: "'Menlo', 'DejaVu Sans Mono', 'Liberation Mono', 'Consolas', 'Ubuntu Mono', 'Courier New'," +
          " 'andale mono', 'lucida console', monospace",
      },
      '& pre code': {
        background: 'none',
        border: 0,
        margin: 0,
        borderRadius: 0,
        display: 'inline',
        padding: 0,
      },
      '& div:not([class]):not([role])': {
        display: 'block',
        marginBlockStart: '1em',
        marginBlockEnd: '1em',
        marginInlineStart: 0,
        marginInlineEnd: 0,
        //animation: `$typing 0.5s steps(30, end), $blinkCaret 0.75s step-end infinite`,
      },
      '& .MuiExpansionPanel-root div': {
        marginBlockStart: 0,
        marginBlockEnd: 0,
      },
    },
    loading: {
      display: 'block',
      marginBlockStart: '1em',
      marginBlockEnd: '1em',
      marginInlineStart: 0,
      marginInlineEnd: 0,
      fontSize: 14,
      color: colors.pgaiDarkGray,
      '&:after': {
        overflow: 'hidden',
        display: 'inline-block',
        verticalAlign: 'bottom',
        animation: '$ellipsis steps(4,end) 1.2s infinite',
        content: "'\\2026'",
        width: 0,
      }
    },
    '@keyframes ellipsis': {
      'to': {
        width: '0.9em'
      },
    },
    '@keyframes typing': {
      from: { width: 0 },
      to: { width: '100%' },
    },
    '@keyframes blinkCaret': {
      from: { borderRightColor: 'transparent' },
      to: { borderRightColor: 'transparent' },
      '50%': { borderRightColor: 'black' },
    },
  }),
)

export const Message = React.memo((props: MessageProps) => {
  const {
    id,
    isAi,
    formattedTime,
    content,
    name,
    created_at,
    isLoading,
    aiModel,
    stateMessage,
    isCurrentStreamMessage,
    isPublic,
    threadId,
    status
  } = props;

  const { updateMessageStatus } = useAiBot()

  const elementRef = useRef<HTMLDivElement | null>(null);


  const [isDebugVisible, setDebugVisible] = useState(false);


  const classes = useStyles();

  useEffect(() => {
    if (!isAi || isCurrentStreamMessage || status === 'read') return;

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0];
        if (entry.isIntersecting && threadId && id) {
          updateMessageStatus(threadId, id, 'read');
          observer.disconnect();
        }
      },
      { threshold: 0.1 }
    );

    if (elementRef.current) {
      observer.observe(elementRef.current);
    }

    return () => {
      observer.disconnect();
    };
  }, [id, updateMessageStatus, isCurrentStreamMessage, isAi, threadId, status]);

  const contentToRender = useMemo(() => {
    if (!content) return '';
    return transformAllCustomTags(content?.replace(/\n/g, '  \n'));
  }, [content]);

  const toggleDebugDialog = () => {
    setDebugVisible(prevState => !prevState)
  }


  const renderers = useMemo<Components>(() => ({
    p: ({ node, ...props }) => <div {...props} />,
    img: ({ node, ...props }) => <img style={{ maxWidth: '60%' }} {...props} />,
    code: ({ node, inline, className, children, ...props }) => {
      const match = /language-(\w+)/.exec(className || '');
      const matchMermaid = /language-mermaid/.test(className || '');
      if (!inline) {
        return (
          <>
            {matchMermaid && !isCurrentStreamMessage && <MermaidDiagram chart={String(children).replace(/\n$/, '')} />}
            <CodeBlock value={String(children).replace(/\n$/, '')} language={match?.[1]} />
          </>
        )
      } else {
        return <code {...props}>{children}</code>
      }
    },
    toolcall: ToolCallRenderer,
    thinkblock: ThinkBlockRenderer,
  }), []);

  return (
    <>
    {id && <DebugDialog
        isOpen={isDebugVisible}
        onClose={toggleDebugDialog}
        messageId={id}
      />}
      <div ref={elementRef} className={classes.message}>
        <div className={classes.messageAvatar}>
          {isAi
            ? <img
              src="/images/bot_avatar.png"
              alt="Postgres.AI Assistant avatar"
              className={classes.messageAvatarImage}
            />
            : icons.userChatIcon}
        </div>
        <MessageHeader
          name={name}
          createdAt={created_at}
          formattedTime={formattedTime}
          id={id}
          isPublic={isPublic}
          isAi={isAi}
          isLoading={isLoading}
          toggleDebugDialog={toggleDebugDialog}
          aiModel={aiModel}
        />
        <div>
          {isLoading
            ?
            <div className={classes.markdown}>
              <div className={classes.loading}>
                {stateMessage && stateMessage.state ? stateMessage.state : 'Thinking'}
              </div>
            </div>
            : <>
                <ReactMarkdown
                  className={classes.markdown}
                  children={contentToRender || ''}
                  rehypePlugins={isAi ? [rehypeRaw] : []}
                  remarkPlugins={[remarkGfm]}
                  linkTarget='_blank'
                  components={renderers}
                  disallowedElements={disallowedHtmlTagsForMarkdown}
                  unwrapDisallowed
                />
                {stateMessage && stateMessage.state && <div className={classes.loading}>
                  {stateMessage.state}
                </div>}
              </>
          }
        </div>
      </div>
    </>
  )
})